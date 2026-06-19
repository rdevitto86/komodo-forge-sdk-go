package logger

import (
	"bytes"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	asyncQueueSize  = 4096
	sinkQueueSize   = 4096
	sinkBatchBytes  = 64 * 1024
	sinkFlushPeriod = time.Second
	sinkTimeout     = 10 * time.Second
	sinkMaxAttempts = 3
)

type fanout struct {
	writers []io.Writer
}

func (f *fanout) Write(p []byte) (int, error) {
	for _, w := range f.writers {
		w.Write(p)
	}
	return len(p), nil
}

type asyncWriter struct {
	dst    io.Writer
	queue  chan *bytes.Buffer
	flushC chan chan struct{}
	done   chan struct{}
	pool   sync.Pool
	wg     sync.WaitGroup
	once   sync.Once
}

func newAsyncWriter(dst io.Writer, size int) *asyncWriter {
	w := &asyncWriter{
		dst:    dst,
		queue:  make(chan *bytes.Buffer, size),
		flushC: make(chan chan struct{}),
		done:   make(chan struct{}),
		pool:   sync.Pool{New: func() any { return new(bytes.Buffer) }},
	}
	w.wg.Add(1)
	go w.run()
	return w
}

func (w *asyncWriter) Write(p []byte) (int, error) {
	b := w.pool.Get().(*bytes.Buffer)
	b.Reset()
	b.Write(p)
	w.queue <- b
	return len(p), nil
}

func (w *asyncWriter) run() {
	defer w.wg.Done()
	defer close(w.done)
	for {
		select {
		case b, ok := <-w.queue:
			if !ok {
				return
			}
			w.dst.Write(b.Bytes())
			w.pool.Put(b)
		case ack := <-w.flushC:
			w.drain()
			close(ack)
		}
	}
}

func (w *asyncWriter) drain() {
	for {
		select {
		case b := <-w.queue:
			w.dst.Write(b.Bytes())
			w.pool.Put(b)
		default:
			return
		}
	}
}

func (w *asyncWriter) Flush() {
	ack := make(chan struct{})
	select {
	case w.flushC <- ack:
		<-ack
	case <-w.done:
	}
}

func (w *asyncWriter) Close() {
	w.once.Do(func() {
		close(w.queue)
		w.wg.Wait()
	})
}

type httpSink struct {
	url     string
	headers map[string]string
	client  *http.Client
	queue   chan *bytes.Buffer
	pool    sync.Pool
	wg      sync.WaitGroup
	once    sync.Once
}

func newHTTPSink(s Sink) *httpSink {
	h := &httpSink{
		url:     s.URL,
		headers: s.Headers,
		client:  &http.Client{Timeout: sinkTimeout},
		queue:   make(chan *bytes.Buffer, sinkQueueSize),
		pool:    sync.Pool{New: func() any { return new(bytes.Buffer) }},
	}
	h.wg.Add(1)
	go h.run()
	return h
}

func (h *httpSink) Write(p []byte) (int, error) {
	b := h.pool.Get().(*bytes.Buffer)
	b.Reset()
	b.Write(p)
	select {
	case h.queue <- b:
	default:
		h.pool.Put(b)
	}
	return len(p), nil
}

func (h *httpSink) run() {
	defer h.wg.Done()
	ticker := time.NewTicker(sinkFlushPeriod)
	defer ticker.Stop()
	batch := new(bytes.Buffer)
	for {
		select {
		case b, ok := <-h.queue:
			if !ok {
				h.post(batch)
				return
			}
			batch.Write(b.Bytes())
			h.pool.Put(b)
			if batch.Len() >= sinkBatchBytes {
				h.post(batch)
			}
		case <-ticker.C:
			h.post(batch)
		}
	}
}

func (h *httpSink) post(batch *bytes.Buffer) {
	if batch.Len() == 0 {
		return
	}
	body := make([]byte, batch.Len())
	copy(body, batch.Bytes())
	batch.Reset()

	for attempt := range sinkMaxAttempts {
		req, err := http.NewRequest(http.MethodPost, h.url, bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		for k, v := range h.headers {
			req.Header.Set(k, v)
		}
		resp, err := h.client.Do(req)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return
			}
		}
		time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
	}
}

func (h *httpSink) Close() {
	h.once.Do(func() {
		close(h.queue)
		h.wg.Wait()
	})
}
