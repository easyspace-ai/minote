package langgraphcompat

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type uniRTFunc func(*http.Request) (*http.Response, error)

func (fn uniRTFunc) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func TestVolcengineUnidirectionalAggregatesBase64Audio(t *testing.T) {
	previousClient := gatewayTTSHTTPClient
	gatewayTTSHTTPClient = &http.Client{
		Transport: uniRTFunc(func(r *http.Request) (*http.Response, error) {
			if got := r.Header.Get("x-api-key"); got != "k" {
				t.Fatalf("x-api-key=%q", got)
			}
			if got := r.Header.Get("X-Api-Resource-Id"); got != "seed-tts-2.0" {
				t.Fatalf("resource-id=%q", got)
			}
			body := strings.Join([]string{
				`{"code":0,"message":"","data":"aGVs"}`,
				`{"code":0,"message":"","data":"bG8="}`,
				`{"code":20000000,"message":"OK","data":null}`,
			}, "\n")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	t.Cleanup(func() { gatewayTTSHTTPClient = previousClient })

	cfg := gatewayTTSConfig{
		VolcAPIKey:     "k",
		VolcVoice:      "seed-tts-2.0",
		VolcHTTPURL:    "https://openspeech.bytedance.com/api/v3/tts/unidirectional",
		VolcResourceID: "seed-tts-2.0",
	}
	audio, format, err := synthesizeVolcengineV3Unidirectional(t.Context(), cfg, gatewayTTSRequest{
		Text:   "你好",
		Format: "mp3",
	})
	if err != nil {
		t.Fatalf("synthesize error: %v", err)
	}
	if string(audio) != "hello" {
		t.Fatalf("audio=%q", string(audio))
	}
	if format != "mp3" {
		t.Fatalf("format=%q", format)
	}
}
