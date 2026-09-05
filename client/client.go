// Package client sends JSON notifications to an Apprise /notify/{key} HTTP API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultTimeout     = 15 * time.Second
	defaultMaxRetries  = 3
	defaultRetryDelay  = 500 * time.Millisecond
	defaultPayloadType = "info"
	defaultFormat      = "text"
)

// Config controls HTTP delivery to Apprise.
type Config struct {
	UserAgent  string
	Timeout    time.Duration
	MaxRetries int
	RetryDelay time.Duration
}

// DefaultConfig returns delivery defaults with the given User-Agent header value.
func DefaultConfig(userAgent string) Config {
	return Config{
		UserAgent:  strings.TrimSpace(userAgent),
		Timeout:    defaultTimeout,
		MaxRetries: defaultMaxRetries,
		RetryDelay: defaultRetryDelay,
	}
}

// FormatUserAgent builds an app/version User-Agent string for Apprise requests.
func FormatUserAgent(appName, version string) string {
	appName = strings.TrimSpace(appName)
	if appName == "" {
		appName = "app"
	}
	version = strings.TrimSpace(version)
	if version == "" {
		version = "dev"
	}
	return appName + "/" + version
}

// Payload is the JSON body accepted by Apprise /notify/{key} endpoints.
type Payload struct {
	Title  string `json:"title,omitempty"`
	Body   string `json:"body"`
	Type   string `json:"type,omitempty"`
	Format string `json:"format,omitempty"`
	Tag    string `json:"tag,omitempty"`
}

type logResponse struct {
	Error   *string    `json:"error"`
	Details [][]string `json:"details"`
}

type statusHandler func([]byte) error

var statusHandlers = map[int]statusHandler{
	http.StatusOK:                          jsonErrorField,
	http.StatusNoContent:                   statusNoContent,
	http.StatusFailedDependency:            func(body []byte) error { return appriseErrorFromBody(body, "one or more notifications could not be sent") },
	http.StatusBadRequest:                  func(body []byte) error { return appriseErrorFromBody(body, "invalid notification request") },
	http.StatusNotAcceptable:               func([]byte) error { return fmt.Errorf("apprise: recursion limit reached (HTTP 406)") },
	http.StatusRequestHeaderFieldsTooLarge: func([]byte) error { return fmt.Errorf("apprise: request payload too large (HTTP 431)") },
}

// ValidateNotifyURL reports whether raw targets a persistent Apprise configuration key.
func ValidateNotifyURL(raw string) error {
	key, ok := persistentNotifyKey(raw)
	if !ok {
		return fmt.Errorf("apprise URL must include a configuration key (e.g. http://apprise:8000/notify/mykey); bare /notify requires urls in each request, which armature-apprise does not send")
	}
	if key == "" {
		return fmt.Errorf("apprise URL missing configuration key after /notify/")
	}
	return nil
}

func persistentNotifyKey(raw string) (string, bool) {
	path, ok := notifyURLPath(raw)
	if !ok {
		return "", false
	}
	return keyAfterNotifyMarker(path)
}

func notifyURLPath(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	return parsedNotifyPath(raw)
}

func parsedNotifyPath(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	return validNotifyPath(strings.TrimSuffix(u.Path, "/"))
}

func validNotifyPath(path string) (string, bool) {
	if path == "" || path == "/notify" || strings.HasSuffix(path, "/notify") {
		return "", false
	}
	return path, true
}

func keyAfterNotifyMarker(path string) (string, bool) {
	const marker = "/notify/"
	idx := strings.LastIndex(path, marker)
	if idx < 0 {
		return "", false
	}
	key := path[idx+len(marker):]
	if key == "" || strings.Contains(key, "/") {
		return "", false
	}
	return key, true
}

// Notify POSTs payload to notifyURL with retries. Empty notifyURL is a no-op.
func Notify(cfg Config, httpClient *http.Client, notifyURL string, payload Payload) error {
	notifyURL = strings.TrimSpace(notifyURL)
	if notifyURL == "" {
		return nil
	}
	if err := ValidateNotifyURL(notifyURL); err != nil {
		return err
	}
	httpClient = clientOrDefault(httpClient, cfg.Timeout)
	payload = normalizePayload(payload)
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal apprise payload: %w", err)
	}
	return notifyWithRetries(cfg, httpClient, notifyURL, body)
}

func clientOrDefault(httpClient *http.Client, timeout time.Duration) *http.Client {
	if httpClient != nil {
		return httpClient
	}
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &http.Client{Timeout: timeout}
}

func normalizePayload(payload Payload) Payload {
	if payload.Type == "" {
		payload.Type = defaultPayloadType
	}
	if payload.Format == "" {
		payload.Format = defaultFormat
	}
	return payload
}

func notifyWithRetries(cfg Config, httpClient *http.Client, notifyURL string, body []byte) error {
	maxRetries, retryDelay := retrySettings(cfg)
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		sleepBeforeRetry(attempt, retryDelay)
		if err := postOnce(cfg, httpClient, notifyURL, body); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return fmt.Errorf("apprise notify failed after %d attempts: %w", maxRetries, lastErr)
}

func retrySettings(cfg Config) (maxRetries int, retryDelay time.Duration) {
	maxRetries = cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = defaultMaxRetries
	}
	retryDelay = cfg.RetryDelay
	if retryDelay <= 0 {
		retryDelay = defaultRetryDelay
	}
	return maxRetries, retryDelay
}

func sleepBeforeRetry(attempt int, retryDelay time.Duration) {
	if attempt > 0 {
		time.Sleep(retryDelay * (1 << (attempt - 1)))
	}
}

func postOnce(cfg Config, httpClient *http.Client, notifyURL string, body []byte) error {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, notifyURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	setRequestHeaders(req, cfg.UserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if readErr != nil {
		return readErr
	}
	return statusError(resp.StatusCode, respBody)
}

func setRequestHeaders(req *http.Request, userAgent string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if ua := strings.TrimSpace(userAgent); ua != "" {
		req.Header.Set("User-Agent", ua)
	}
}

func statusError(statusCode int, respBody []byte) error {
	if handler, ok := statusHandlers[statusCode]; ok {
		return handler(respBody)
	}
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}
	return unexpectedStatusError(statusCode, respBody)
}

func statusNoContent([]byte) error {
	return errNoTargetsMatched()
}

func errNoTargetsMatched() error {
	return fmt.Errorf("apprise: no notification targets matched (HTTP 204); verify the configuration key exists in Apprise and tag filters match configured services")
}

func unexpectedStatusError(statusCode int, respBody []byte) error {
	if msg := strings.TrimSpace(string(respBody)); msg != "" {
		return fmt.Errorf("apprise: HTTP %d: %s", statusCode, msg)
	}
	return fmt.Errorf("apprise: HTTP %d", statusCode)
}

func jsonErrorField(respBody []byte) error {
	return parsedAppriseError(respBody)
}

func appriseErrorFromBody(respBody []byte, fallback string) error {
	if err := parsedAppriseError(respBody); err != nil {
		return err
	}
	if msg := strings.TrimSpace(string(respBody)); msg != "" {
		return fmt.Errorf("apprise: %s", msg)
	}
	return fmt.Errorf("apprise: %s", fallback)
}

func parsedAppriseError(respBody []byte) error {
	if len(respBody) == 0 {
		return nil
	}
	var parsed logResponse
	if err := json.Unmarshal(respBody, &parsed); err == nil {
		if parsed.Error != nil && strings.TrimSpace(*parsed.Error) != "" {
			return fmt.Errorf("apprise: %s", strings.TrimSpace(*parsed.Error))
		}
	}
	return nil
}
