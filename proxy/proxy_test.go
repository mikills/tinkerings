package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestProxy(t *testing.T) {
	t.Run("basic GET request", func(t *testing.T) {
		// create a test backend server
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Backend", "true")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "backend response")
		}))
		defer backend.Close()

		// create proxy
		target, _ := url.Parse(backend.URL)
		p, err := New(&Config{TargetURL: target})
		if err != nil {
			t.Fatalf("failed to create proxy: %v", err)
		}

		// make request through proxy
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		p.ServeHTTP(rec, req)

		// verify response
		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		if rec.Body.String() != "backend response" {
			t.Errorf("expected body 'backend response', got %s", rec.Body.String())
		}

		if rec.Header().Get("X-Backend") != "true" {
			t.Error("expected X-Backend header to be true")
		}
	})

	t.Run("POST with body", func(t *testing.T) {
		// create a test backend server that echoes the body
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
			w.Write(body)
		}))
		defer backend.Close()

		// create proxy
		target, _ := url.Parse(backend.URL)
		p, err := New(&Config{TargetURL: target})
		if err != nil {
			t.Fatalf("failed to create proxy: %v", err)
		}

		// make request through proxy
		reqBody := "test request body"
		req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
		rec := httptest.NewRecorder()

		p.ServeHTTP(rec, req)

		// verify response
		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		if rec.Body.String() != reqBody {
			t.Errorf("expected body '%s', got %s", reqBody, rec.Body.String())
		}
	})

	t.Run("preserves query parameters", func(t *testing.T) {
		// create a test backend server
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			fmt.Fprintf(w, "param1=%s,param2=%s", query.Get("param1"), query.Get("param2"))
		}))
		defer backend.Close()

		// create proxy
		target, _ := url.Parse(backend.URL)
		p, err := New(&Config{TargetURL: target})
		if err != nil {
			t.Fatalf("failed to create proxy: %v", err)
		}

		// make request through proxy
		req := httptest.NewRequest("GET", "/test?param1=value1&param2=value2", nil)
		rec := httptest.NewRecorder()

		p.ServeHTTP(rec, req)

		// verify response
		expected := "param1=value1,param2=value2"
		if rec.Body.String() != expected {
			t.Errorf("expected body '%s', got %s", expected, rec.Body.String())
		}
	})

	t.Run("filters hop-by-hop headers", func(t *testing.T) {
		// create a test backend server that echoes headers
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// check that hop-by-hop headers are not present
			if r.Header.Get("Connection") != "" {
				t.Error("Connection header should be filtered")
			}
			if r.Header.Get("Keep-Alive") != "" {
				t.Error("Keep-Alive header should be filtered")
			}
			// check that normal headers are preserved
			if r.Header.Get("X-Custom") != "value" {
				t.Error("X-Custom header should be preserved")
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer backend.Close()

		// create proxy
		target, _ := url.Parse(backend.URL)
		p, err := New(&Config{TargetURL: target})
		if err != nil {
			t.Fatalf("failed to create proxy: %v", err)
		}

		// make request through proxy with hop-by-hop headers
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Keep-Alive", "timeout=5")
		req.Header.Set("X-Custom", "value")
		rec := httptest.NewRecorder()

		p.ServeHTTP(rec, req)
	})

	t.Run("adds X-Forwarded headers", func(t *testing.T) {
		// create a test backend server that checks forwarded headers
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// write back the forwarded headers for verification
			fmt.Fprintf(w, "X-Forwarded-For=%s,X-Forwarded-Host=%s,X-Forwarded-Proto=%s",
				r.Header.Get("X-Forwarded-For"),
				r.Header.Get("X-Forwarded-Host"),
				r.Header.Get("X-Forwarded-Proto"))
		}))
		defer backend.Close()

		// create proxy
		target, _ := url.Parse(backend.URL)
		p, err := New(&Config{TargetURL: target})
		if err != nil {
			t.Fatalf("failed to create proxy: %v", err)
		}

		// make request through proxy
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		req.Host = "original.host.com"
		rec := httptest.NewRecorder()

		p.ServeHTTP(rec, req)

		// verify forwarded headers were added
		body := rec.Body.String()
		if !strings.Contains(body, "X-Forwarded-For=192.168.1.1") {
			t.Errorf("expected X-Forwarded-For to contain client IP, got: %s", body)
		}
		if !strings.Contains(body, "X-Forwarded-Host=original.host.com") {
			t.Errorf("expected X-Forwarded-Host to contain original host, got: %s", body)
		}
		if !strings.Contains(body, "X-Forwarded-Proto=http") {
			t.Errorf("expected X-Forwarded-Proto to be http, got: %s", body)
		}
	})

	t.Run("request modifier", func(t *testing.T) {
		// create a test backend server
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, r.Header.Get("X-Modified"))
		}))
		defer backend.Close()

		// create proxy with request modifier
		target, _ := url.Parse(backend.URL)
		p, err := New(&Config{
			TargetURL: target,
			ModifyRequest: func(r *http.Request) error {
				r.Header.Set("X-Modified", "true")
				return nil
			},
		})
		if err != nil {
			t.Fatalf("failed to create proxy: %v", err)
		}

		// make request through proxy
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		p.ServeHTTP(rec, req)

		// verify request was modified
		if rec.Body.String() != "true" {
			t.Errorf("expected modified header to be set, got: %s", rec.Body.String())
		}
	})

	t.Run("response modifier", func(t *testing.T) {
		// create a test backend server
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Original", "true")
			fmt.Fprint(w, "original")
		}))
		defer backend.Close()

		// create proxy with response modifier
		target, _ := url.Parse(backend.URL)
		p, err := New(&Config{
			TargetURL: target,
			ModifyResponse: func(resp *http.Response) error {
				resp.Header.Set("X-Modified-Response", "true")
				return nil
			},
		})
		if err != nil {
			t.Fatalf("failed to create proxy: %v", err)
		}

		// make request through proxy
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		p.ServeHTTP(rec, req)

		// verify response was modified
		if rec.Header().Get("X-Modified-Response") != "true" {
			t.Error("expected modified response header to be set")
		}
		if rec.Header().Get("X-Original") != "true" {
			t.Error("expected original header to be preserved")
		}
	})

	t.Run("error handling - request modifier error", func(t *testing.T) {
		// create a test backend server
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer backend.Close()

		// track if error handler was called
		errorHandlerCalled := false

		// create proxy with failing request modifier
		target, _ := url.Parse(backend.URL)
		p, err := New(&Config{
			TargetURL: target,
			ModifyRequest: func(r *http.Request) error {
				return fmt.Errorf("request modification failed")
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				errorHandlerCalled = true
				http.Error(w, err.Error(), http.StatusInternalServerError)
			},
		})
		if err != nil {
			t.Fatalf("failed to create proxy: %v", err)
		}

		// make request through proxy
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		p.ServeHTTP(rec, req)

		// verify error was handled
		if !errorHandlerCalled {
			t.Error("expected error handler to be called")
		}
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rec.Code)
		}
	})

	t.Run("handles backend errors", func(t *testing.T) {
		// create proxy pointing to non-existent backend
		target, _ := url.Parse("http://localhost:0")

		errorHandlerCalled := false
		p, err := New(&Config{
			TargetURL: target,
			Timeout:   100 * time.Millisecond,
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				errorHandlerCalled = true
				http.Error(w, "Backend unavailable", http.StatusBadGateway)
			},
		})
		if err != nil {
			t.Fatalf("failed to create proxy: %v", err)
		}

		// make request through proxy
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		p.ServeHTTP(rec, req)

		// verify error was handled
		if !errorHandlerCalled {
			t.Error("expected error handler to be called for backend error")
		}
		if rec.Code != http.StatusBadGateway {
			t.Errorf("expected status 502, got %d", rec.Code)
		}
	})

	t.Run("handles various HTTP methods", func(t *testing.T) {
		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

		// create a test backend server that echoes the method
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, r.Method)
		}))
		defer backend.Close()

		// create proxy
		target, _ := url.Parse(backend.URL)
		p, err := New(&Config{TargetURL: target})
		if err != nil {
			t.Fatalf("failed to create proxy: %v", err)
		}

		for _, method := range methods {
			t.Run(method, func(t *testing.T) {
				var body io.Reader
				if method != "GET" && method != "HEAD" {
					body = strings.NewReader("test body")
				}

				req := httptest.NewRequest(method, "/test", body)
				rec := httptest.NewRecorder()

				p.ServeHTTP(rec, req)

				// HEAD responses don't have body
				if method != "HEAD" {
					if rec.Body.String() != method {
						t.Errorf("expected method %s to be echoed, got %s", method, rec.Body.String())
					}
				}
			})
		}
	})

	t.Run("preserves status codes", func(t *testing.T) {
		statusCodes := []int{200, 201, 301, 400, 404, 500}

		for _, code := range statusCodes {
			t.Run(fmt.Sprintf("status_%d", code), func(t *testing.T) {
				// create a test backend server that returns specific status
				backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(code)
				}))
				defer backend.Close()

				// create proxy
				target, _ := url.Parse(backend.URL)
				p, err := New(&Config{TargetURL: target})
				if err != nil {
					t.Fatalf("failed to create proxy: %v", err)
				}

				req := httptest.NewRequest("GET", "/test", nil)
				rec := httptest.NewRecorder()

				p.ServeHTTP(rec, req)

				if rec.Code != code {
					t.Errorf("expected status %d, got %d", code, rec.Code)
				}
			})
		}
	})

	t.Run("handles large payloads", func(t *testing.T) {
		// create a large payload (1MB)
		largeData := bytes.Repeat([]byte("a"), 1024*1024)

		// create a test backend server that echoes the body
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			w.Write(body)
		}))
		defer backend.Close()

		// create proxy
		target, _ := url.Parse(backend.URL)
		p, err := New(&Config{TargetURL: target})
		if err != nil {
			t.Fatalf("failed to create proxy: %v", err)
		}

		// make request through proxy
		req := httptest.NewRequest("POST", "/test", bytes.NewReader(largeData))
		rec := httptest.NewRecorder()

		p.ServeHTTP(rec, req)

		// verify response
		if !bytes.Equal(rec.Body.Bytes(), largeData) {
			t.Errorf("large payload was not correctly proxied, got %d bytes, expected %d",
				rec.Body.Len(), len(largeData))
		}
	})

	t.Run("handles concurrent requests", func(t *testing.T) {
		// create a test backend server with request counter
		var requestCount int32
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&requestCount, 1)
			time.Sleep(10 * time.Millisecond) // simulate some processing
			fmt.Fprintf(w, "request_%d", atomic.LoadInt32(&requestCount))
		}))
		defer backend.Close()

		// create proxy
		target, _ := url.Parse(backend.URL)
		p, err := New(&Config{TargetURL: target})
		if err != nil {
			t.Fatalf("failed to create proxy: %v", err)
		}

		// make concurrent requests
		const numRequests = 50
		var wg sync.WaitGroup
		wg.Add(numRequests)

		for i := 0; i < numRequests; i++ {
			go func(id int) {
				defer wg.Done()

				req := httptest.NewRequest("GET", fmt.Sprintf("/test_%d", id), nil)
				rec := httptest.NewRecorder()

				p.ServeHTTP(rec, req)

				if rec.Code != http.StatusOK {
					t.Errorf("request %d failed with status %d", id, rec.Code)
				}
			}(i)
		}

		wg.Wait()

		// verify all requests were _claim
		if atomic.LoadInt32(&requestCount) != numRequests {
			t.Errorf("expected %d requests, got %d", numRequests, requestCount)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		// create a slow backend server
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-time.After(5 * time.Second):
				fmt.Fprint(w, "slow response")
			case <-r.Context().Done():
				// request was cancelled
				return
			}
		}))
		defer backend.Close()

		// create proxy
		target, _ := url.Parse(backend.URL)
		p, err := New(&Config{TargetURL: target})
		if err != nil {
			t.Fatalf("failed to create proxy: %v", err)
		}

		// create request with cancellable context
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		errorHandled := false
		p.config.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			errorHandled = true
			http.Error(w, "Request timeout", http.StatusGatewayTimeout)
		}

		p.ServeHTTP(rec, req)

		// verify request was cancelled
		if !errorHandled {
			t.Error("expected error handler to be called for cancelled context")
		}
		if rec.Code != http.StatusGatewayTimeout {
			t.Errorf("expected status 504, got %d", rec.Code)
		}
	})
}

func TestSingleJoiningSlash(t *testing.T) {
	tests := []struct {
		a, b     string
		expected string
	}{
		{"", "", "/"},
		{"/", "", "/"},
		{"", "/", "/"},
		{"/", "/", "/"},
		{"/api", "/v1", "/api/v1"},
		{"/api/", "v1", "/api/v1"},
		{"/api", "v1", "/api/v1"},
		{"/api/", "/v1", "/api/v1"},
		{"api", "v1", "api/v1"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s+%s", tt.a, tt.b), func(t *testing.T) {
			result := singleJoiningSlash(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("singleJoiningSlash(%q, %q) = %q, want %q",
					tt.a, tt.b, result, tt.expected)
			}
		})
	}
}
