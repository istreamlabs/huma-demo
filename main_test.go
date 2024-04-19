package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
)

var _ huma.ResolverWithPath = (*VideoEncoder)(nil)

const channelFixture = `{
  "name": "test channel",
  "on": true,
  "publish_points": [
    {
      "id": "pub1",
			"format": "hls",
			"drms": ["fairplay"],
      "url": "http://example.com"
    }
  ],
  "region": "us-west",
  "segment_duration": 6,
  "video_encoders": [
    {
      "bitrate": 2000,
      "framerate": 30,
      "height": 1080,
      "id": "hd",
      "width": 1920
    }
  ]
}`

// expectStatus is a helper to check/assert the status code of a response.
func expectStatus(tb testing.TB, resp *httptest.ResponseRecorder, code int) {
	tb.Helper()
	if resp.Code != code {
		tb.Fatalf("expected status %d, got %d", code, resp.Code)
	}
}

type testWriter struct {
	tb testing.TB
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	w.tb.Log(string(p))
	return len(p), nil
}

func TestAPI(t *testing.T) {
	// Write logs using `t.Log` instead of to stdout.
	slog.SetDefault(slog.New(slog.NewTextHandler(&testWriter{tb: t}, nil)))

	_, api := humatest.New(t)

	db := NewDB[*ChannelMeta]("")

	setup(api, db)

	var ch Channel
	if err := json.Unmarshal([]byte(channelFixture), &ch); err != nil {
		t.Fatalf("failed to unmarshal channel fixture: %s", err)
	}

	resp := api.Put("/channels/test", &ch)
	expectStatus(t, resp, http.StatusNoContent)

	resp = api.Put("/channels/test2", &ch)
	expectStatus(t, resp, http.StatusNoContent)
	etag := resp.Header().Get("ETag")

	// Fail conditional update
	resp = api.Put("/channels/test2", "If-Match: badvalue", &ch)
	expectStatus(t, resp, http.StatusPreconditionFailed)

	// Try again with correct ETag hash value
	ch.Name = "updated channel"
	resp = api.Put("/channels/test2", "If-Match: "+etag, &ch)
	expectStatus(t, resp, http.StatusNoContent)

	// Fail with bad value
	ch.VideoEncoders[0].Width = 1600
	resp = api.Put("/channels/test2", &ch)
	expectStatus(t, resp, http.StatusUnprocessableEntity)

	resp = api.Get("/channels")
	expectStatus(t, resp, http.StatusOK)
	var v []any
	if err := json.Unmarshal(resp.Body.Bytes(), &v); err != nil {
		t.Fatalf("failed to unmarshal response: %s", err)
	}
	if len(v) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(v))
	}

	resp = api.Get("/channels/test")
	expectStatus(t, resp, http.StatusOK)
	if !strings.Contains(resp.Body.String(), "test channel") {
		t.Fatalf("expected response to contain channel name, got %s", resp.Body.String())
	}

	resp = api.Get("/channels/missing")
	expectStatus(t, resp, http.StatusNotFound)

	resp = api.Delete("/channels/test")
	expectStatus(t, resp, http.StatusNoContent)
}
