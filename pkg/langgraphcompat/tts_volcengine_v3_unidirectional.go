package langgraphcompat

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type volcUniResp struct {
	Code    int     `json:"code"`
	Message string  `json:"message"`
	Data    *string `json:"data"`
}

func synthesizeVolcengineV3Unidirectional(ctx context.Context, cfg gatewayTTSConfig, req gatewayTTSRequest) ([]byte, string, error) {
	input := strings.TrimSpace(firstNonEmpty(req.Text, req.Input))
	if input == "" {
		return nil, "", fmt.Errorf("volcengine v3 tts: empty input")
	}
	format := normalizeTTSFormat(firstNonEmpty(req.Format, req.ResponseFmt))
	voice := strings.TrimSpace(firstNonEmpty(req.Voice, cfg.VolcVoice))
	if voice == "" {
		return nil, "", fmt.Errorf("volcengine v3 tts: missing voice type")
	}

	audioParams := map[string]any{
		"format":      volcV3Format(format),
		"sample_rate": 24000,
	}
	if req.SpeedRatio > 0 {
		audioParams["speed_ratio"] = req.SpeedRatio
	}
	if req.VolumeRatio > 0 {
		audioParams["volume_ratio"] = req.VolumeRatio
	}
	if req.PitchRatio > 0 {
		audioParams["pitch_ratio"] = req.PitchRatio
	}

	payload := map[string]any{
		"req_params": map[string]any{
			"text":    input,
			"speaker": voice,
			"additions": `{"disable_markdown_filter":true,"enable_language_detector":true,"enable_latex_tn":true,` +
				`"disable_default_bit_rate":true,"max_length_to_filter_parenthesis":0,` +
				`"cache_config":{"text_type":1,"use_cache":true}}`,
			"audio_params": audioParams,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.VolcHTTPURL, bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	// ByteDance 文档/官方示例使用小写 x-api-key（与 cmd/volc-tts 一致）。
	httpReq.Header.Set("x-api-key", strings.TrimSpace(cfg.VolcAPIKey))
	httpReq.Header.Set("X-Api-Resource-Id", firstNonEmpty(cfg.VolcResourceID, volcV3ResourceIDByVoice(voice)))
	httpReq.Header.Set("Connection", "keep-alive")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "*/*")

	resp, err := gatewayTTSHTTPClient.Do(httpReq)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, "", fmt.Errorf("volcengine v3 unidirectional http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	dec := json.NewDecoder(resp.Body)
	var audio bytes.Buffer
	lastCode := 0
	lastMsg := ""
	for {
		var pkt volcUniResp
		if err := dec.Decode(&pkt); err != nil {
			if err == io.EOF {
				break
			}
			return nil, "", fmt.Errorf("volcengine v3 unidirectional decode: %w", err)
		}
		lastCode = pkt.Code
		lastMsg = strings.TrimSpace(pkt.Message)
		if pkt.Code != 0 && pkt.Code != 20000000 {
			if lastMsg == "" {
				lastMsg = "unknown error"
			}
			return nil, "", fmt.Errorf("volcengine v3 unidirectional code=%d: %s", pkt.Code, lastMsg)
		}
		if pkt.Data != nil && strings.TrimSpace(*pkt.Data) != "" {
			chunk, err := base64.StdEncoding.DecodeString(*pkt.Data)
			if err != nil {
				return nil, "", fmt.Errorf("volcengine v3 unidirectional base64 decode: %w", err)
			}
			if len(chunk) > 0 {
				_, _ = audio.Write(chunk)
			}
		}
	}
	if audio.Len() == 0 {
		return nil, "", fmt.Errorf("volcengine v3 unidirectional empty audio (last code=%d message=%q)", lastCode, lastMsg)
	}
	return audio.Bytes(), format, nil
}

func volcV3ResourceIDByVoice(voice string) string {
	v := strings.TrimSpace(voice)
	low := strings.ToLower(v)
	if strings.HasPrefix(v, "S_") {
		return "volc.megatts.default"
	}
	if strings.HasPrefix(low, "seed-tts") {
		return v
	}
	if strings.Contains(low, "seed_tts") {
		return v
	}
	return "volc.service_type.10029"
}

func volcV3Format(format string) string {
	switch normalizeTTSFormat(format) {
	case "wav":
		return "wav"
	case "pcm":
		return "pcm"
	default:
		return "mp3"
	}
}
