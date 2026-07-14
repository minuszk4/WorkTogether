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
	"strings"
	"testing"
)

func TestGoogleTranslatorUsesServiceAccountAndTranslationAPI(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if r.Form.Get("assertion") == "" {
				t.Fatal("service-account assertion was not sent")
			}
			_, _ = w.Write([]byte(`{"access_token":"test-token","expires_in":3600}`))
		case "/language/translate/v2":
			if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
				t.Fatalf("Authorization = %q", got)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if got := r.Form.Get("q"); got != "hello" {
				t.Fatalf("q = %q", got)
			}
			if got := r.Form.Get("target"); got != "vi" {
				t.Fatalf("target = %q", got)
			}
			_, _ = w.Write([]byte(`{"data":{"translations":[{"translatedText":"xin ch&#224;o","detectedSourceLanguage":"en"}]}}`))
		default:
			t.Fatalf("unexpected request path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	credentials, err := json.Marshal(map[string]string{
		"client_email": "translator@example.test",
		"private_key":  string(keyPEM),
		"token_uri":    server.URL + "/token",
		"project_id":   "test-project",
	})
	if err != nil {
		t.Fatal(err)
	}
	translator, err := NewGoogleTranslatorFromJSON(credentials, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	translator.endpoint = server.URL + "/language/translate/v2"

	result, err := translator.Translate(context.Background(), Job{Text: "hello", SourceLanguage: "en", TargetLanguage: "vi"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "xin chào" || result.DetectedLanguage != "en" {
		t.Fatalf("unexpected translation: %#v", result)
	}
	if strings.TrimSpace(result.TargetLanguage) != "vi" {
		t.Fatalf("target language = %q", result.TargetLanguage)
	}
}
