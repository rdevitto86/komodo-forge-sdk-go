package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

func tokenServer(t *testing.T, fetches *int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(fetches, 1)
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		if got := r.Form.Get("grant_type"); got != "client_credentials" {
			t.Errorf("grant_type = %q, want client_credentials", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tokenResponse{AccessToken: "tok-123", TokenType: "Bearer", ExpiresIn: 3600})
	}))
}

func TestClientCredentials_RequiresConfig(t *testing.T) {
	if _, err := NewClientCredentialsTokenSource(ServiceAuthConfig{ClientID: "a", ClientSecret: "b"}); err == nil {
		t.Error("expected error when TokenURL is empty")
	}
	if _, err := NewClientCredentialsTokenSource(ServiceAuthConfig{TokenURL: "http://x"}); err == nil {
		t.Error("expected error when credentials are empty")
	}
}

func TestClientCredentials_FetchAndCache(t *testing.T) {
	var fetches int32
	srv := tokenServer(t, &fetches)
	defer srv.Close()

	src, err := NewClientCredentialsTokenSource(ServiceAuthConfig{
		TokenURL: srv.URL, ClientID: "svc", ClientSecret: "secret", Scope: "svc:order-api",
	})
	if err != nil {
		t.Fatalf("NewClientCredentialsTokenSource: %v", err)
	}

	for range 5 {
		tok, err := src.Token(context.Background())
		if err != nil {
			t.Fatalf("Token: %v", err)
		}
		if tok != "tok-123" {
			t.Fatalf("token = %q, want tok-123", tok)
		}
	}
	if got := atomic.LoadInt32(&fetches); got != 1 {
		t.Errorf("fetches = %d, want 1 (token should be cached)", got)
	}
}

func TestClientCredentials_DedupesConcurrentFetch(t *testing.T) {
	var fetches int32
	srv := tokenServer(t, &fetches)
	defer srv.Close()

	src, err := NewClientCredentialsTokenSource(ServiceAuthConfig{
		TokenURL: srv.URL, ClientID: "svc", ClientSecret: "secret",
	})
	if err != nil {
		t.Fatalf("NewClientCredentialsTokenSource: %v", err)
	}

	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			if _, err := src.Token(context.Background()); err != nil {
				t.Errorf("Token: %v", err)
			}
		})
	}
	wg.Wait()
	if got := atomic.LoadInt32(&fetches); got != 1 {
		t.Errorf("fetches = %d, want 1 (concurrent fetches should collapse)", got)
	}
}

func TestClientCredentials_ErrorOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	src, _ := NewClientCredentialsTokenSource(ServiceAuthConfig{
		TokenURL: srv.URL, ClientID: "svc", ClientSecret: "bad",
	})
	if _, err := src.Token(context.Background()); err == nil {
		t.Error("expected error on 401 from token endpoint")
	}
}

func TestWithServiceAuth_AttachesBearer(t *testing.T) {
	var fetches int32
	tsrv := tokenServer(t, &fetches)
	defer tsrv.Close()

	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	src, _ := NewClientCredentialsTokenSource(ServiceAuthConfig{
		TokenURL: tsrv.URL, ClientID: "svc", ClientSecret: "secret",
	})
	httpClient := &http.Client{Transport: WithServiceAuth(nil, src)}

	req, _ := http.NewRequest(http.MethodGet, upstream.URL, nil)
	res, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	res.Body.Close()

	if gotAuth != "Bearer tok-123" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer tok-123")
	}
	// The original request must not have been mutated by the round-tripper.
	if req.Header.Get("Authorization") != "" {
		t.Error("round-tripper mutated the inbound request")
	}
}
