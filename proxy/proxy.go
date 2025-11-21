package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RequestModifier allows modification of outgoing requests
type RequestModifier func(*http.Request) error

// ResponseModifier allows modification of responses before sending back
type ResponseModifier func(*http.Response) error

type Config struct {
	// target URL to proxy requests to
	TargetURL *url.URL

	// timeout for the proxy client
	Timeout time.Duration

	// custom transport for the proxy client
	Transport http.RoundTripper

	// modify request before sending to target
	ModifyRequest RequestModifier

	// modify response before sending back to client
	ModifyResponse ResponseModifier

	// error handler for proxy errors
	ErrorHandler func(http.ResponseWriter, *http.Request, error)
}

// Proxy represents an HTTP reverse proxy
type Proxy struct {
	config *Config
	client *http.Client
}

// DefaultConfig returns a Config with sensible defaults
// Only the TargetURL needs to be set after calling this
func DefaultConfig() *Config {
	return &Config{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConnsPerHost:   10,
		},
		ErrorHandler: defaultErrorHandler,
	}
}

// New creates a new proxy instance with the given configuration
func New(config *Config) (*Proxy, error) {
	if config.TargetURL == nil {
		return nil, fmt.Errorf("target URL is required")
	}

	// apply defaults for any unset fields
	defaults := DefaultConfig()

	if config.Timeout == 0 {
		config.Timeout = defaults.Timeout
	}

	if config.Transport == nil {
		config.Transport = defaults.Transport
	}

	if config.ErrorHandler == nil {
		config.ErrorHandler = defaults.ErrorHandler
	}

	return &Proxy{
		config: config,
		client: &http.Client{
			Transport: config.Transport,
			Timeout:   config.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // don't follow redirects
			},
		},
	}, nil
}

// NewWithDefaults creates a new proxy with default configuration
func NewWithDefaults(targetURL string) (*Proxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL: %w", err)
	}

	config := DefaultConfig()
	config.TargetURL = target

	return New(config)
}

// ServeHTTP handles incoming requests and proxies them to the target
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// create the proxied request
	proxyReq, err := p.createProxyRequest(ctx, r)
	if err != nil {
		p.config.ErrorHandler(w, r, fmt.Errorf("failed to create proxy request: %w", err))
		return
	}

	// apply request modifier if configured
	if p.config.ModifyRequest != nil {
		if err := p.config.ModifyRequest(proxyReq); err != nil {
			p.config.ErrorHandler(w, r, fmt.Errorf("request modifier failed: %w", err))
			return
		}
	}

	// execute the request
	resp, err := p.client.Do(proxyReq)
	if err != nil {
		p.config.ErrorHandler(w, r, fmt.Errorf("proxy request failed: %w", err))
		return
	}
	defer resp.Body.Close()

	// apply response modifier if configured
	if p.config.ModifyResponse != nil {
		if err := p.config.ModifyResponse(resp); err != nil {
			p.config.ErrorHandler(w, r, fmt.Errorf("response modifier failed: %w", err))
			return
		}
	}

	// copy response headers
	p.copyHeaders(w.Header(), resp.Header)

	// set status code
	w.WriteHeader(resp.StatusCode)

	// copy response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		// can't really handle this error as headers are already sent
		_ = err
	}
}

// createProxyRequest creates a new request to be sent to the target
func (p *Proxy) createProxyRequest(ctx context.Context, r *http.Request) (*http.Request, error) {
	// build target URL
	targetURL := *p.config.TargetURL
	targetURL.Path = singleJoiningSlash(targetURL.Path, r.URL.Path)
	targetURL.RawQuery = r.URL.RawQuery

	// create new request with the same method and body
	proxyReq, err := http.NewRequestWithContext(ctx, r.Method, targetURL.String(), r.Body)
	if err != nil {
		return nil, err
	}

	// copy headers, filtering out hop-by-hop headers
	p.copyHeaders(proxyReq.Header, r.Header)

	// set X-Forwarded headers
	if clientIP, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		appendHeader(proxyReq.Header, "X-Forwarded-For", clientIP)
	}
	appendHeader(proxyReq.Header, "X-Forwarded-Host", r.Host)

	// set the host header to the target host
	proxyReq.Host = p.config.TargetURL.Host

	// preserve the original protocol
	if r.TLS != nil {
		appendHeader(proxyReq.Header, "X-Forwarded-Proto", "https")
	} else {
		appendHeader(proxyReq.Header, "X-Forwarded-Proto", "http")
	}

	// close notification
	proxyReq.Close = false

	return proxyReq, nil
}

// copyHeaders copies headers from src to dst, filtering out hop-by-hop headers
func (p *Proxy) copyHeaders(dst, src http.Header) {
	for name, values := range src {
		// skip hop-by-hop headers
		if isHopHeader(name) {
			continue
		}

		for _, value := range values {
			dst.Add(name, value)
		}
	}
}

// isHopHeader checks if a header is a hop-by-hop header
func isHopHeader(name string) bool {
	name = strings.ToLower(name)
	for _, h := range hopHeaders {
		if strings.ToLower(h) == name {
			return true
		}
	}
	return false
}

// appendHeader appends a value to a header if it doesn't already exist
func appendHeader(header http.Header, name, value string) {
	if current := header.Get(name); current != "" {
		header.Set(name, current+", "+value)
	} else {
		header.Set(name, value)
	}
}

// singleJoiningSlash joins two paths with a single slash
func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")

	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// defaultErrorHandler writes a generic error response
func defaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, "Bad Gateway", http.StatusBadGateway)
}

// Handler returns an http.Handler for the proxy
func (p *Proxy) Handler() http.Handler {
	return http.HandlerFunc(p.ServeHTTP)
}

// hop-by-hop headers that should not be forwarded
var hopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}
