package owrap

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync/atomic"
	"time"
)

type AuthorisedCallback func() (string, error)

type ProxyConfig struct {
	Target       *url.URL
	APIKey       atomic.Value // stores string
	AuthCallback AuthorisedCallback
}

type Proxy struct {
	config *ProxyConfig
	proxy  *httputil.ReverseProxy
}

func NewProxy(cfg *ProxyConfig) *Proxy {
	if cfg.AuthCallback == nil {
		cfg.AuthCallback = func() (string, error) { return "", nil }
	}
	// initialize atomic value if unset
	if _, ok := cfg.APIKey.Load().(string); !ok {
		cfg.APIKey.Store("")
	}

	proxy := httputil.NewSingleHostReverseProxy(cfg.Target)
	orig := proxy.Director

	proxy.Director = func(req *http.Request) {
		orig(req)
		req.Host = cfg.Target.Host
		cfg.ApplyAuthorization(req)
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Error("proxy error", "error", err)
		http.Error(w, "upstream error", http.StatusBadGateway)
	}

	proxy.ModifyResponse = func(res *http.Response) error {
		// clear api key on unauthorized so the next request refetches
		if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
			cfg.APIKey.Store("")
			slog.Warn("cleared api key after unauthorized response", "status", res.StatusCode)
			return nil
		}
		// no callback invocation here; api key retrieval happens lazily in ApplyAuthorization
		return nil
	}

	return &Proxy{
		config: cfg,
		proxy:  proxy,
	}
}

func (cfg *ProxyConfig) ApplyAuthorization(req *http.Request) {
	req.Header.Del("Authorization")
	apiKey, _ := cfg.APIKey.Load().(string)
	if apiKey == "" {
		key, err := cfg.AuthCallback()
		if err != nil {
			slog.Error("api key callback failed", "error", err)
		} else if key != "" {
			cfg.APIKey.Store(key)
			apiKey = key
			slog.Info("stored api key")
		}
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
}
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.proxy.ServeHTTP(w, r)
}

func NewServer(listenAddr string, proxy *Proxy) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/", loggingMiddleware(proxy))

	return &http.Server{
		Addr:           listenAddr,
		Handler:        mux,
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   0, // no timeout for streaming responses
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(lrw, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "status", lrw.statusCode, "duration", time.Since(start))
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lw *loggingResponseWriter) WriteHeader(code int) {
	lw.statusCode = code
	lw.ResponseWriter.WriteHeader(code)
}
