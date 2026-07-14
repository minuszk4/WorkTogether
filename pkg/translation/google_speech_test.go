package translation

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGoogleSpeechTranscriberSendsWebMAndReturnsTranscript(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			_, _ = w.Write([]byte(`{"access_token":"test-token","expires_in":3600}`))
		case "/v1/speech:recognize":
			if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
				t.Fatalf("Authorization = %q", got)
			}
			var request struct {
				Config struct {
					Encoding string `json:"encoding"`
					Language string `json:"languageCode"`
				} `json:"config"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.Config.Encoding != "WEBM_OPUS" || request.Config.Language != "vi-VN" {
				t.Fatalf("unexpected config: %#v", request.Config)
			}
			_, _ = w.Write([]byte(`{"results":[{"alternatives":[{"transcript":"xin chào mọi người"}]}]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	credentials, err := json.Marshal(map[string]string{
		"client_email": "speech@example.test",
		"private_key":  string(keyPEM),
		"token_uri":    server.URL + "/token",
	})
	if err != nil {
		t.Fatal(err)
	}
	transcriber, err := NewGoogleSpeechTranscriberFromJSON(credentials, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	transcriber.endpoint = server.URL + "/v1/speech:recognize"

	text, err := transcriber.Transcribe(context.Background(), []byte("webm audio"), "vi-VN")
	if err != nil {
		t.Fatal(err)
	}
	if text != "xin chào mọi người" {
		t.Fatalf("transcript = %q", text)
	}
}
