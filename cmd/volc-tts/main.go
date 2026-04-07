package main

import (
	"bytes"
	"encoding/base64"
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

type streamResp struct {
	Code    int     `json:"code"`
	Message string  `json:"message"`
	Data    *string `json:"data"`
}

func main() {
	dotenv.Load()

	endpoint := flag.String("endpoint", firstNonEmpty(os.Getenv("VOLCENGINE_TTS_HTTP_ENDPOINT"), "https://openspeech.bytedance.com/api/v3/tts/unidirectional"), "Volcengine TTS unidirectional endpoint")
	apiKey := flag.String("api-key", firstNonEmpty(os.Getenv("VOLCENGINE_TTS_API_KEY"), os.Getenv("TTS_API_KEY")), "Volcengine API key (x-api-key)")
	resourceID := flag.String("resource-id", firstNonEmpty(os.Getenv("VOLCENGINE_TTS_RESOURCE_ID"), "volc.service_type.10029"), "X-Api-Resource-Id")
	voice := flag.String("voice", firstNonEmpty(os.Getenv("VOLCENGINE_TTS_VOICE_TYPE"), os.Getenv("TTS_VOICE"), "zh_male_beijingxiaoye_emo_v2_mars_bigtts"), "Speaker/voice type")
	text := flag.String("text", "豆包语音", "Text to synthesize")
	format := flag.String("format", "mp3", "Audio format: mp3|wav|pcm")
	out := flag.String("out", "", "Output file path (default: ./tmp/volc-tts-demo.<format>)")
	timeout := flag.Duration("timeout", 180*time.Second, "Request timeout")
	flag.Parse()

	if strings.TrimSpace(*apiKey) == "" {
		fatalf("missing api key: set -api-key or VOLCENGINE_TTS_API_KEY")
	}

	reqBody := map[string]any{
		"req_params": map[string]any{
			"text":    strings.TrimSpace(*text),
			"speaker": strings.TrimSpace(*voice),
			"additions": `{"disable_markdown_filter":true,"enable_language_detector":true,"enable_latex_tn":true,` +
				`"disable_default_bit_rate":true,"max_length_to_filter_parenthesis":0,` +
				`"cache_config":{"text_type":1,"use_cache":true}}`,
			"audio_params": map[string]any{
				"format":      normalizeFormat(*format),
				"sample_rate": 24000,
			},
		},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		fatalf("marshal request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, strings.TrimSpace(*endpoint), bytes.NewReader(payload))
	if err != nil {
		fatalf("new request: %v", err)
	}
	req.Header.Set("x-api-key", strings.TrimSpace(*apiKey))
	req.Header.Set("X-Api-Resource-Id", strings.TrimSpace(*resourceID))
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")

	client := &http.Client{Timeout: *timeout}
	resp, err := client.Do(req)
	if err != nil {
		fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		fatalf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	dec := json.NewDecoder(resp.Body)
	var audio bytes.Buffer
	for {
		var evt streamResp
		if err := dec.Decode(&evt); err != nil {
			if err == io.EOF {
				break
			}
			fatalf("decode response: %v", err)
		}
		if evt.Code != 0 && evt.Code != 20000000 {
			fatalf("volc error code=%d message=%s", evt.Code, strings.TrimSpace(evt.Message))
		}
		if evt.Data != nil && strings.TrimSpace(*evt.Data) != "" {
			chunk, err := base64.StdEncoding.DecodeString(*evt.Data)
			if err != nil {
				fatalf("decode base64 chunk: %v", err)
			}
			_, _ = audio.Write(chunk)
		}
	}

	if audio.Len() == 0 {
		fatalf("empty audio in stream response")
	}
	output := strings.TrimSpace(*out)
	if output == "" {
		output = filepath.Join("tmp", "volc-tts-demo."+normalizeFormat(*format))
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		fatalf("mkdir output dir: %v", err)
	}
	if err := os.WriteFile(output, audio.Bytes(), 0o644); err != nil {
		fatalf("write output: %v", err)
	}
	fmt.Printf("Volc TTS OK\n")
	fmt.Printf("  Endpoint: %s\n", *endpoint)
	fmt.Printf("  Bytes:    %d\n", audio.Len())
	fmt.Printf("  Output:   %s\n", output)
}

func normalizeFormat(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "wav", "pcm":
		return strings.ToLower(strings.TrimSpace(s))
	default:
		return "mp3"
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "volc-tts-demo: "+format+"\n", args...)
	os.Exit(1)
}
