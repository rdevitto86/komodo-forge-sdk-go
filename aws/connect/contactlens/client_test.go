package contactlens

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connectcontactlens"
	clstypes "github.com/aws/aws-sdk-go-v2/service/connectcontactlens/types"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

type fakeContactLensAPI struct {
	capturedIn *connectcontactlens.ListRealtimeContactAnalysisSegmentsInput
	out        *connectcontactlens.ListRealtimeContactAnalysisSegmentsOutput
	err        error
}

func (f *fakeContactLensAPI) ListRealtimeContactAnalysisSegments(_ context.Context, in *connectcontactlens.ListRealtimeContactAnalysisSegmentsInput, _ ...func(*connectcontactlens.Options)) (*connectcontactlens.ListRealtimeContactAnalysisSegmentsOutput, error) {
	f.capturedIn = in
	return f.out, f.err
}

// ── Unit Tests ───────────────────────────────────────────────────────────────

func TestNew_MissingRegion(t *testing.T) {
	_, err := New(context.Background(), Config{})
	if err == nil {
		t.Fatal("expected error for missing region, got nil")
	}
	if err == nil || err.Error() != "missing region" {
		t.Errorf("got %v, want \"missing region\"", err)
	}
}

func TestListRealtimeContactAnalysisSegments_HappyPath(t *testing.T) {
	begin := int32(0)
	end := int32(2500)

	fake := &fakeContactLensAPI{
		out: &connectcontactlens.ListRealtimeContactAnalysisSegmentsOutput{
			Segments: []clstypes.RealtimeContactAnalysisSegment{
				{
					Transcript: &clstypes.Transcript{
						Content:           aws.String("Hello, how can I help?"),
						BeginOffsetMillis: &begin,
						EndOffsetMillis:   &end,
						ParticipantId:     aws.String("AGENT"),
						Sentiment:         clstypes.SentimentValueNeutral,
					},
				},
			},
		},
	}
	c := newWithAPI(fake)

	segments, err := c.ListRealtimeContactAnalysisSegments(context.Background(), "inst-1", "contact-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(segments) != 1 {
		t.Fatalf("len(segments) = %d, want 1", len(segments))
	}

	seg := segments[0]
	if seg.Type != "TRANSCRIPT" {
		t.Errorf("Type = %q, want %q", seg.Type, "TRANSCRIPT")
	}
	if seg.Content != "Hello, how can I help?" {
		t.Errorf("Content = %q, want %q", seg.Content, "Hello, how can I help?")
	}
	if seg.BeginOffsetMillis != 0 {
		t.Errorf("BeginOffsetMillis = %d, want 0", seg.BeginOffsetMillis)
	}
	if seg.EndOffsetMillis != 2500 {
		t.Errorf("EndOffsetMillis = %d, want 2500", seg.EndOffsetMillis)
	}
	if seg.ParticipantID != "AGENT" {
		t.Errorf("ParticipantID = %q, want %q", seg.ParticipantID, "AGENT")
	}
	if seg.Sentiment != string(clstypes.SentimentValueNeutral) {
		t.Errorf("Sentiment = %q, want %q", seg.Sentiment, string(clstypes.SentimentValueNeutral))
	}

	// Verify input translation.
	if aws.ToString(fake.capturedIn.InstanceId) != "inst-1" {
		t.Errorf("InstanceId = %q, want %q", aws.ToString(fake.capturedIn.InstanceId), "inst-1")
	}
	if aws.ToString(fake.capturedIn.ContactId) != "contact-1" {
		t.Errorf("ContactId = %q, want %q", aws.ToString(fake.capturedIn.ContactId), "contact-1")
	}
}

func TestListRealtimeContactAnalysisSegments_SDKError(t *testing.T) {
	sdkErr := errors.New("contact lens unavailable")
	fake := &fakeContactLensAPI{err: sdkErr}
	c := newWithAPI(fake)

	_, err := c.ListRealtimeContactAnalysisSegments(context.Background(), "inst-1", "contact-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, sdkErr) {
		t.Errorf("error chain does not wrap SDK error: %v", err)
	}
}

func TestListRealtimeContactAnalysisSegments_ValidationErrors(t *testing.T) {
	c := newWithAPI(&fakeContactLensAPI{})

	cases := []struct {
		name       string
		instanceID string
		contactID  string
	}{
		{"missing instanceID", "", "contact-1"},
		{"missing contactID", "inst-1", ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := c.ListRealtimeContactAnalysisSegments(context.Background(), tc.instanceID, tc.contactID)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}
