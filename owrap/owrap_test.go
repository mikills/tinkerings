package owrap

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestProxy_InjectsAuthHeader(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-api-key" {
			t.Errorf("expected Authorization header 'Bearer test-api-key', got %q", auth)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream response"))
	}))
	defer upstream.Close()

	targetURL, _ := url.Parse(upstream.URL)
	cfg := &ProxyConfig{
		Target: targetURL,
		AuthCallback: func() (string, error) {
			return "test-api-key", nil
		},
	}
	proxy := NewProxy(cfg)

	// test request with different auth header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer client-key")
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body, _ := io.ReadAll(w.Body)
	if string(body) != "upstream response" {
		t.Errorf("expected 'upstream response', got %q", string(body))
	}
}

func TestProxy_CallbackOnSuccess(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	callbackCalled := false

	targetURL, _ := url.Parse(upstream.URL)
	cfg := &ProxyConfig{
		Target: targetURL,
		AuthCallback: func() (string, error) {
			if !callbackCalled {
				callbackCalled = true
			}
			return "test-key", nil
		},
	}
	proxy := NewProxy(cfg)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	if !callbackCalled {
		t.Error("callback was not called")
	}

	if !callbackCalled {
		t.Error("expected api key callback to run")
	}
}

func TestProxy_NoCallbackOnError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorised"))
	}))
	defer upstream.Close()

	callbackCalled := false

	targetURL, _ := url.Parse(upstream.URL)
	cfg := &ProxyConfig{
		Target: targetURL,
		AuthCallback: func() (string, error) {
			if !callbackCalled {
				callbackCalled = true
			}
			return "test-key", nil
		},
	}
	cfg.APIKey.Store("test-key")
	proxy := NewProxy(cfg)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	if callbackCalled {
		t.Error("callback should not be called on 401 response")
	}

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
	if k, _ := cfg.APIKey.Load().(string); k != "" {
		t.Errorf("expected api key to be cleared after unauthorized, still have %q", k)
	}
}
