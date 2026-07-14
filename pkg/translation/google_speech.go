package translation

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const maxCaptionAudioBytes = 128 * 1024

type GoogleSpeechTranscriber struct {
	google   *GoogleTranslator
	endpoint string
}

func NewGoogleSpeechTranscriber(credentialsPath string, client *http.Client) (*GoogleSpeechTranscriber, error) {
	credentials, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("read Google credentials: %w", err)
	}
	return NewGoogleSpeechTranscriberFromJSON(credentials, client)
}

func NewGoogleSpeechTranscriberFromJSON(credentials []byte, client *http.Client) (*GoogleSpeechTranscriber, error) {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	google, err := NewGoogleTranslatorFromJSON(credentials, client)
	if err != nil {
		return nil, err
	}
	return &GoogleSpeechTranscriber{google: google, endpoint: "https://speech.googleapis.com/v1/speech:recognize"}, nil
}

func (s *GoogleSpeechTranscriber) Transcribe(ctx context.Context, audio []byte, language string) (string, error) {
	if len(audio) == 0 || len(audio) > maxCaptionAudioBytes {
		return "", fmt.Errorf("caption audio must be between 1 and %d bytes", maxCaptionAudioBytes)
	}
	if language == "" {
		language = "en-US"
	}
	token, err := s.google.token(ctx)
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(struct {
		Config struct {
			Encoding                   string `json:"encoding"`
			LanguageCode               string `json:"languageCode"`
			EnableAutomaticPunctuation bool   `json:"enableAutomaticPunctuation"`
		} `json:"config"`
		Audio struct {
			Content string `json:"content"`
		} `json:"audio"`
	}{
		Config: struct {
			Encoding                   string `json:"encoding"`
			LanguageCode               string `json:"languageCode"`
			EnableAutomaticPunctuation bool   `json:"enableAutomaticPunctuation"`
		}{Encoding: "WEBM_OPUS", LanguageCode: language, EnableAutomaticPunctuation: true},
		Audio: struct {
			Content string `json:"content"`
		}{Content: base64.StdEncoding.EncodeToString(audio)},
	})
	if err != nil {
		return "", fmt.Errorf("encode speech request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("create speech request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	response, err := s.google.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request speech recognition: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return "", fmt.Errorf("speech API returned %s: %s", response.Status, strings.TrimSpace(string(responseBody)))
	}
	var payload struct {
		Results []struct {
			Alternatives []struct {
				Transcript string `json:"transcript"`
			} `json:"alternatives"`
		} `json:"results"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode speech response: %w", err)
	}
	parts := make([]string, 0, len(payload.Results))
	for _, result := range payload.Results {
		if len(result.Alternatives) > 0 && strings.TrimSpace(result.Alternatives[0].Transcript) != "" {
			parts = append(parts, strings.TrimSpace(result.Alternatives[0].Transcript))
		}
	}
	return strings.Join(parts, " "), nil
}
