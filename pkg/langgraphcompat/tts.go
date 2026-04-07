package langgraphcompat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

var gatewayTTSHTTPClient = &http.Client{Timeout: 2 * time.Minute}

// gatewayTTSRequest mirrors POST /api/tts JSON (Studio 与 cmd/tts 共用)。
type gatewayTTSRequest struct {
	Text         string  `json:"text"`
	Model        string  `json:"model,omitempty"`
	Voice        string  `json:"voice,omitempty"`
	Format       string  `json:"format,omitempty"`
	Instructions string  `json:"instructions,omitempty"`
	SpeedRatio   float64 `json:"speed_ratio,omitempty"`
	VolumeRatio  float64 `json:"volume_ratio,omitempty"`
	PitchRatio   float64 `json:"pitch_ratio,omitempty"`
	Input        string  `json:"input,omitempty"`
	ResponseFmt  string  `json:"response_format,omitempty"`
}

// gatewayTTSConfig is Volcengine 豆包 openspeech 单向 HTTP（与 cmd/volc-tts 一致）。
type gatewayTTSConfig struct {
	VolcAPIKey     string
	VolcVoice      string
	VolcHTTPURL    string
	VolcResourceID string
}

func (s *Server) handleTTS(w http.ResponseWriter, r *http.Request) {
	var req gatewayTTSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid request body"})
		return
	}

	input := strings.TrimSpace(firstNonEmpty(req.Text, req.Input))
	if input == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "text is required"})
		return
	}

	cfg, err := gatewayVolcTTSConfigFromEnv()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"detail": err.Error()})
		return
	}

	audio, format, err := synthesizeVolcengineV3Unidirectional(r.Context(), cfg, req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"detail": err.Error()})
		return
	}

	w.Header().Set("Content-Type", ttsContentType(format))
	w.Header().Set("Content-Disposition", contentDisposition("attachment", "speech."+format))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(audio)
}

func gatewayVolcTTSConfigFromEnv() (gatewayTTSConfig, error) {
	apiKey := strings.TrimSpace(os.Getenv("VOLCENGINE_TTS_API_KEY"))
	token := strings.TrimSpace(os.Getenv("VOLCENGINE_TTS_ACCESS_TOKEN"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("TTS_API_KEY"))
	}
	appID := strings.TrimSpace(os.Getenv("VOLCENGINE_TTS_APP_ID"))
	if apiKey == "" && token != "" {
		apiKey = token
	}
	if apiKey == "" && strings.HasPrefix(strings.ToLower(appID), "api-key-") {
		apiKey = appID
	}
	voice := "zh_male_beijingxiaoye_emo_v2_mars_bigtts"
	httpURL := strings.TrimSpace(os.Getenv("VOLCENGINE_TTS_HTTP_ENDPOINT"))
	if httpURL == "" {
		httpURL = "https://openspeech.bytedance.com/api/v3/tts/unidirectional"
	}

	if apiKey == "" {
		return gatewayTTSConfig{}, fmt.Errorf(
			"tts: set VOLCENGINE_TTS_API_KEY or TTS_API_KEY (火山 openspeech x-api-key)",
		)
	}
	return gatewayTTSConfig{
		VolcAPIKey:     apiKey,
		VolcVoice:      voice,
		VolcHTTPURL:    strings.TrimRight(httpURL, "/"),
		VolcResourceID: strings.TrimSpace(os.Getenv("VOLCENGINE_TTS_RESOURCE_ID")),
	}, nil
}

func normalizeTTSFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "mp3", "opus", "aac", "flac", "wav", "pcm":
		return strings.ToLower(strings.TrimSpace(format))
	default:
		return "mp3"
	}
}

func ttsContentType(format string) string {
	switch normalizeTTSFormat(format) {
	case "wav":
		return "audio/wav"
	case "flac":
		return "audio/flac"
	case "aac":
		return "audio/aac"
	case "opus":
		return "audio/opus"
	case "pcm":
		return "audio/pcm"
	default:
		return "audio/mpeg"
	}
}
