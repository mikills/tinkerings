package owrap

import (
	"log/slog"
	"net/http"
	"net/url"
	"os"
)

func main() {
	upstreamURL := os.Getenv("UPSTREAM_URL")
	apiKey := os.Getenv("UPSTREAM_API_KEY")
	listenAddr := getEnv("LISTEN_ADDR", ":8080")

	if upstreamURL == "" {
		slog.Error("missing upstream url", "env", "UPSTREAM_URL")
		os.Exit(1)
	}
	if apiKey == "" {
		slog.Error("missing api key", "env", "UPSTREAM_API_KEY")
		os.Exit(1)
	}

	target, err := url.Parse(upstreamURL)
	if err != nil {
		slog.Error("invalid upstream url", "value", upstreamURL, "error", err)
		os.Exit(1)
	}

	// callback returning api key
	authorisedCb := func() (string, error) {
		return apiKey, nil
	}

	cfg := &ProxyConfig{
		Target:       target,
		AuthCallback: authorisedCb,
	}
	cfg.APIKey.Store(apiKey)
	proxy := NewProxy(cfg)

	server := NewServer(listenAddr, proxy)

	slog.Info("proxy listening", "addr", listenAddr, "upstream", upstreamURL)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
