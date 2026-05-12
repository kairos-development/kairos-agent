package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"
)

var (
	// ErrDisabled is returned when no cloud base URL is configured.
	ErrDisabled = errors.New("cloud integration is disabled")
	// ErrUnauthorized is returned when the cloud rejects the bearer token.
	ErrUnauthorized = errors.New("cloud request unauthorized")
	// ErrNoUpdateAvailable is returned when cloud reports no matching release.
	ErrNoUpdateAvailable = errors.New("no update available")
)

// Config describes cloud transport settings. AccessToken must come from vault,
// environment, or another secret provider, not from plaintext config files.
type Config struct {
	BaseURL     string
	AccessToken string
	UserAgent   string
	Timeout     time.Duration
	HTTPClient  *http.Client
}

// Client is a small HTTP adapter for optional Kairos Cloud APIs.
type Client struct {
	baseURL     string
	accessToken string
	userAgent   string
	httpClient  *http.Client
}

// HandshakeInput is the cloud-visible agent identity and heartbeat envelope.
type HandshakeInput struct {
	OrganizationID string                 `json:"organization_id"`
	Fingerprint    string                 `json:"fingerprint"`
	Name           string                 `json:"name"`
	Platform       string                 `json:"platform"`
	AgentVersion   string                 `json:"agent_version"`
	Status         string                 `json:"status"`
	TelemetryOptIn bool                   `json:"telemetry_opt_in"`
	Metrics        map[string]interface{} `json:"metrics,omitempty"`
}

// HandshakeOutput contains cloud authorization hints. It must not be used as a
// substitute for local risk checks or local recovery decisions.
type HandshakeOutput struct {
	DeviceID     string                 `json:"device_id"`
	TrustLevel   string                 `json:"trust_level"`
	Entitlements map[string]interface{} `json:"entitlements"`
	TelemetryOn  bool                   `json:"telemetry_on"`
}

// UpdateQuery identifies the target artifact for a signed update check.
type UpdateQuery struct {
	CurrentVersion string
	Channel        string
	OS             string
	Arch           string
}

// UpdateResponse contains signed update metadata returned by Kairos Cloud.
type UpdateResponse struct {
	UpdateAvailable bool             `json:"update_available"`
	Release         *ReleaseMetadata `json:"release,omitempty"`
	Artifact        *Artifact        `json:"artifact,omitempty"`
}

// ReleaseMetadata describes a cloud release without embedding local install logic.
type ReleaseMetadata struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Channel     string `json:"channel"`
	Criticality string `json:"criticality"`
	NotesURL    string `json:"notes_url"`
	PublishedAt string `json:"published_at"`
}

// Artifact describes a signed binary artifact for one OS/architecture pair.
type Artifact struct {
	ID        string `json:"id"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	Signature string `json:"signature"`
	SizeBytes int64  `json:"size_bytes"`
}

// NewClient creates a cloud HTTP client.
func NewClient(cfg Config) (*Client, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		return nil, ErrDisabled
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid cloud base url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("cloud base url must use http or https")
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	} else if httpClient.Timeout == 0 {
		clone := *httpClient
		clone.Timeout = timeout
		httpClient = &clone
	}
	userAgent := cfg.UserAgent
	if userAgent == "" {
		userAgent = "kairos-agent"
	}

	return &Client{baseURL: baseURL, accessToken: cfg.AccessToken, userAgent: userAgent, httpClient: httpClient}, nil
}

// Handshake registers or refreshes an agent device in Kairos Cloud.
func (c *Client) Handshake(ctx context.Context, input HandshakeInput) (HandshakeOutput, error) {
	var out HandshakeOutput
	if input.Platform == "" {
		input.Platform = runtime.GOOS
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/agent/handshake", input, &out); err != nil {
		return HandshakeOutput{}, err
	}
	return out, nil
}

// CheckUpdate asks Kairos Cloud for a signed release matching the target.
func (c *Client) CheckUpdate(ctx context.Context, query UpdateQuery) (UpdateResponse, error) {
	if query.OS == "" {
		query.OS = runtime.GOOS
	}
	if query.Arch == "" {
		query.Arch = runtime.GOARCH
	}
	if query.Channel == "" {
		query.Channel = "stable"
	}
	values := url.Values{}
	values.Set("current_version", query.CurrentVersion)
	values.Set("channel", query.Channel)
	values.Set("os", query.OS)
	values.Set("arch", query.Arch)

	var out UpdateResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/releases/update?"+values.Encode(), nil, &out); err != nil {
		return UpdateResponse{}, err
	}
	if !out.UpdateAvailable {
		return out, ErrNoUpdateAvailable
	}
	return out, nil
}

func (c *Client) doJSON(ctx context.Context, method string, path string, input interface{}, output interface{}) error {
	var body io.Reader
	if input != nil {
		payload, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("encode cloud request: %w", err)
		}
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("create cloud request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cloud request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return ErrUnauthorized
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		limited, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("cloud request returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(limited)))
	}
	if output == nil {
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(output); err != nil {
		return fmt.Errorf("decode cloud response: %w", err)
	}
	return nil
}
