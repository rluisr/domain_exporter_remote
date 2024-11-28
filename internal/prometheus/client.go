package promclient

import (
	"errors"
	"net/http"
	"time"

	"github.com/caarlos0/domain_exporter/internal/safeconfig"
	"github.com/castai/promwrite"
)

func NewClient(cfg safeconfig.SafeConfig) (*promwrite.Client, error) {
	username := cfg.Prometheus.User
	password := cfg.Prometheus.Pass

	if username == "" || password == "" {
		return nil, errors.New("invalid prometheus config")
	}

	basicAuthTransport := &BasicAuthTransport{
		Username: username,
		Password: password,
		Wrapped:  http.DefaultTransport,
	}

	customClient := &http.Client{
		Transport: basicAuthTransport,
		Timeout:   30 * time.Second,
	}

	promClient := promwrite.NewClient(cfg.Prometheus.URL, promwrite.HttpClient(customClient))

	return promClient, nil
}

// BasicAuthTransport は Basic Auth ヘッダーを追加するカスタム Transport です。
type BasicAuthTransport struct {
	Username string
	Password string
	Wrapped  http.RoundTripper
}

// RoundTrip はリクエストに Basic Auth ヘッダーを追加します。
func (t *BasicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(t.Username, t.Password)
	return t.Wrapped.RoundTrip(req)
}
