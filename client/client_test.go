package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConfig() Config {
	return DefaultConfig("armature-apprise/test")
}

func TestNotify_EmptyURLIsNoop(t *testing.T) {
	err := Notify(testConfig(), nil, "", Payload{Title: "t", Body: "b"})
	assert.NoError(t, err)
}

func TestNotify_PostsJSONPayloadWithTag(t *testing.T) {
	var gotMethod string
	var gotContentType string
	var gotAccept string
	var gotUserAgent string
	var gotBody Payload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		gotAccept = r.Header.Get("Accept")
		gotUserAgent = r.Header.Get("User-Agent")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":null,"details":[]}`))
	}))
	defer srv.Close()

	cfg := testConfig()
	err := Notify(cfg, srv.Client(), srv.URL+"/notify/testkey", Payload{
		Title: "Approval",
		Body:  "Please approve",
		Type:  "info",
		Tag:   "starapp_uid_7",
	})
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "application/json", gotContentType)
	assert.Equal(t, "application/json", gotAccept)
	assert.Equal(t, cfg.UserAgent, gotUserAgent)
	assert.Equal(t, "Approval", gotBody.Title)
	assert.Equal(t, "Please approve", gotBody.Body)
	assert.Equal(t, "info", gotBody.Type)
	assert.Equal(t, "text", gotBody.Format)
	assert.Equal(t, "starapp_uid_7", gotBody.Tag)
}

func TestNotify_DefaultsTypeToInfo(t *testing.T) {
	var gotBody Payload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":null,"details":[]}`))
	}))
	defer srv.Close()

	require.NoError(t, Notify(testConfig(), srv.Client(), srv.URL+"/notify/testkey", Payload{Title: "t", Body: "b"}))
	assert.Equal(t, "info", gotBody.Type)
	assert.Equal(t, "text", gotBody.Format)
}

func TestNotify_RetriesOnFailureThenSucceeds(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		if n < 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":null,"details":[]}`))
	}))
	defer srv.Close()

	err := Notify(testConfig(), srv.Client(), srv.URL+"/notify/testkey", Payload{Title: "t", Body: "b"})
	require.NoError(t, err)
	assert.Equal(t, int32(2), attempts.Load())
}

func TestValidateNotifyURL(t *testing.T) {
	require.NoError(t, ValidateNotifyURL("http://apprise:8000/notify/mykey"))
	require.NoError(t, ValidateNotifyURL("https://apprise.example/notify/mykey/"))

	err := ValidateNotifyURL("http://apprise:8000/notify")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "configuration key")

	err = ValidateNotifyURL("http://apprise:8000/notify/")
	require.Error(t, err)
}

func TestNotify_RejectsNoContentAsFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := Notify(testConfig(), srv.Client(), srv.URL+"/notify/testkey", Payload{Title: "t", Body: "b"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 204")
}

func TestNotify_RejectsJSONErrorField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":"delivery failed","details":[]}`))
	}))
	defer srv.Close()

	err := Notify(testConfig(), srv.Client(), srv.URL+"/notify/testkey", Payload{Title: "t", Body: "b"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delivery failed")
}

func TestFormatUserAgent(t *testing.T) {
	assert.Equal(t, "StarApp/1.2.3", FormatUserAgent("StarApp", "1.2.3"))
	assert.Equal(t, "app/dev", FormatUserAgent("", ""))
}
