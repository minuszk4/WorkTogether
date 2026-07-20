package translation

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const googleCloudScope = "https://www.googleapis.com/auth/cloud-platform"

type googleServiceAccount struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
}

// GoogleTranslator calls Cloud Translation with a service account. It does not
// send or store audio; callers pass already-produced text only.
type GoogleTranslator struct {
	credentials googleServiceAccount
	privateKey  *rsa.PrivateKey
	client      *http.Client
	endpoint    string

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

func NewGoogleTranslator(credentialsPath string, client *http.Client) (*GoogleTranslator, error) {
	credentials, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("read Google credentials: %w", err)
	}
	return NewGoogleTranslatorFromJSON(credentials, client)
}

func NewGoogleTranslatorFromJSON(credentials []byte, client *http.Client) (*GoogleTranslator, error) {
	var account googleServiceAccount
	if err := json.Unmarshal(credentials, &account); err != nil {
		return nil, fmt.Errorf("parse Google credentials: %w", err)
	}
	if account.ClientEmail == "" || account.PrivateKey == "" || account.TokenURI == "" {
		return nil, fmt.Errorf("Google credentials are missing client_email, private_key, or token_uri")
	}
	block, _ := pem.Decode([]byte(account.PrivateKey))
	if block == nil {
		return nil, fmt.Errorf("parse Google private key: no PEM block")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		parsed, parseErr := x509.ParsePKCS8PrivateKey(block.Bytes)
		if parseErr != nil {
			return nil, fmt.Errorf("parse Google private key: %w", err)
		}
		var ok bool
		key, ok = parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("parse Google private key: expected RSA key")
		}
	}
	if client == nil {
		client = &http.Client{Timeout: 1500 * time.Millisecond}
	}
	return &GoogleTranslator{
		credentials: account,
		privateKey:  key,
		client:      client,
		endpoint:    "https://translation.googleapis.com/language/translate/v2",
	}, nil
}

func (g *GoogleTranslator) Translate(ctx context.Context, job Job) (Result, error) {
	if strings.TrimSpace(job.Text) == "" || strings.TrimSpace(job.TargetLanguage) == "" {
		return Result{}, fmt.Errorf("translation text and target language are required")
	}
	token, err := g.token(ctx)
	if err != nil {
		return Result{}, err
	}
	values := url.Values{"q": {job.Text}, "target": {job.TargetLanguage}, "format": {"text"}}
	if job.SourceLanguage != "" {
		values.Set("source", job.SourceLanguage)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return Result{}, fmt.Errorf("create translation request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := g.client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("request translation: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return Result{}, fmt.Errorf("translation API returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	var payload struct {
		Data struct {
			Translations []struct {
				Text             string `json:"translatedText"`
				DetectedLanguage string `json:"detectedSourceLanguage"`
			} `json:"translations"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil {
		return Result{}, fmt.Errorf("decode translation response: %w", err)
	}
	if len(payload.Data.Translations) == 0 {
		return Result{}, fmt.Errorf("translation API returned no translation")
	}
	translation := payload.Data.Translations[0]
	return Result{Job: job, Text: html.UnescapeString(translation.Text), DetectedLanguage: translation.DetectedLanguage}, nil
}

func (g *GoogleTranslator) token(ctx context.Context) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.accessToken != "" && time.Until(g.expiresAt) > time.Minute {
		return g.accessToken, nil
	}
	assertion, err := g.signedAssertion(time.Now())
	if err != nil {
		return "", err
	}
	values := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {assertion},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.credentials.TokenURI, strings.NewReader(values.Encode()))
	if err != nil {
		return "", fmt.Errorf("create Google token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request Google access token: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return "", fmt.Errorf("Google token endpoint returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode Google token response: %w", err)
	}
	if payload.AccessToken == "" {
		return "", fmt.Errorf("Google token response has no access token")
	}
	g.accessToken = payload.AccessToken
	g.expiresAt = time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second)
	return g.accessToken, nil
}

func (g *GoogleTranslator) signedAssertion(now time.Time) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	claims, err := json.Marshal(map[string]any{
		"iss":   g.credentials.ClientEmail,
		"scope": googleCloudScope,
		"aud":   g.credentials.TokenURI,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	})
	if err != nil {
		return "", fmt.Errorf("encode Google token claims: %w", err)
	}
	unsigned := header + "." + base64.RawURLEncoding.EncodeToString(claims)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(nil, g.privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign Google token assertion: %w", err)
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}
