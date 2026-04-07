package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/easyspace-ai/minote/pkg/dotenv"
)

type ttsRequest struct {
	Text         string  `json:"text,omitempty"`
	Input        string  `json:"input,omitempty"`
	Format       string  `json:"format,omitempty"`
	ResponseFmt  string  `json:"response_format,omitempty"`
	Voice        string  `json:"voice,omitempty"`
	SpeedRatio   float64 `json:"speed_ratio,omitempty"`
	Instructions string  `json:"instructions,omitempty"`
}

func main() {
	dotenv.Load()

	baseURL := flag.String("base-url", firstNonEmpty(os.Getenv("TTS_DEMO_BASE_URL"), "http://127.0.0.1:8787"), "Gateway base URL")
	path := flag.String("path", "/api/tts", "TTS endpoint path")
	text := flag.String("text", "你好，这是一个 TTS 连通性测试。", "Text to synthesize")
	format := flag.String("format", "mp3", "Audio format: mp3|wav|opus|pcm")
	voice := flag.String("voice", "", "Optional request voice override")
	speed := flag.Float64("speed", 1.0, "Optional speed ratio")
	output := flag.String("out", "", "Output file path (default: ./tmp/tts-demo.<format>)")
	authToken := flag.String("auth-token", strings.TrimSpace(os.Getenv("DEERFLOW_AUTH_TOKEN")), "Optional Bearer token")
	timeout := flag.Duration("timeout", 180*time.Second, "Request timeout")
	flag.Parse()

	reqBody := ttsRequest{
		Text:        strings.TrimSpace(*text),
		Input:       strings.TrimSpace(*text),
		Format:      strings.TrimSpace(*format),
		ResponseFmt: strings.TrimSpace(*format),
	}
	if v := strings.TrimSpace(*voice); v != "" {
		reqBody.Voice = v
	}
	if *speed > 0 {
		reqBody.SpeedRatio = *speed
	}

	if reqBody.Text == "" {
		fatalf("text cannot be empty")
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		fatalf("marshal request: %v", err)
	}

	url := strings.TrimRight(strings.TrimSpace(*baseURL), "/") + ensureLeadingSlash(strings.TrimSpace(*path))
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		fatalf("build request: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", mimeByFormat(reqBody.Format))
	if token := strings.TrimSpace(*authToken); token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: *timeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		fatalf("read response: %v", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fatalf("tts failed: HTTP %d, body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	out := strings.TrimSpace(*output)
	if out == "" {
		out = filepath.Join("tmp", "tts-demo."+normalizeFormat(reqBody.Format))
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		fatalf("mkdir output dir: %v", err)
	}
	if err := os.WriteFile(out, raw, 0o644); err != nil {
		fatalf("write output: %v", err)
	}

	fmt.Printf("TTS OK\n")
	fmt.Printf("  URL:        %s\n", url)
	fmt.Printf("  Backend:    Volcengine openspeech HTTP (configure VOLCENGINE_TTS_API_KEY or TTS_API_KEY)\n")
	fmt.Printf("  Voice(env): %s\n", firstNonEmpty(os.Getenv("VOLCENGINE_TTS_VOICE_TYPE"), os.Getenv("TTS_VOICE"), "(server default)"))
	fmt.Printf("  Bytes:      %d\n", len(raw))
	fmt.Printf("  MIME:       %s\n", firstNonEmpty(resp.Header.Get("Content-Type"), "(unknown)"))
	fmt.Printf("  Output:     %s\n", out)
}

func normalizeFormat(s string) string {
	f := strings.ToLower(strings.TrimSpace(s))
	switch f {
	case "wav", "opus", "pcm":
		return f
	default:
		return "mp3"
	}
}

func mimeByFormat(format string) string {
	switch normalizeFormat(format) {
	case "wav":
		return "audio/wav"
	case "opus":
		return "audio/ogg"
	case "pcm":
		return "audio/L16"
	default:
		return "audio/mpeg"
	}
}

func ensureLeadingSlash(s string) string {
	if s == "" {
		return "/"
	}
	if strings.HasPrefix(s, "/") {
		return s
	}
	return "/" + s
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "tts-demo: "+format+"\n", args...)
	os.Exit(1)
}
