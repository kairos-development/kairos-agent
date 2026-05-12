package cloud

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestNewClient_RejectsDisabledOrInvalidURL(t *testing.T) {
	_, err := NewClient(Config{})
	assert.ErrorIs(t, err, ErrDisabled)

	_, err = NewClient(Config{BaseURL: "ftp://cloud.example"})
	assert.Error(t, err)
}

func TestCheckUpdate_ReturnsSignedMetadata(t *testing.T) {
	client, err := NewClient(Config{
		BaseURL:     "https://cloud.example/",
		AccessToken: "token",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "/api/v1/releases/update", req.URL.Path)
			assert.Equal(t, "0.9.0", req.URL.Query().Get("current_version"))
			assert.Equal(t, "Bearer token", req.Header.Get("Authorization"))
			return jsonResponse(http.StatusOK, `{"update_available":true,"release":{"id":"rel_1","version":"1.0.0","channel":"stable","criticality":"normal","notes_url":"https://example.test/notes","published_at":"2026-05-12T00:00:00Z"},"artifact":{"id":"art_1","os":"linux","arch":"amd64","url":"https://example.test/kairos","sha256":"abc","signature":"sig","size_bytes":42}}`), nil
		})},
	})
	require.NoError(t, err)

	out, err := client.CheckUpdate(context.Background(), UpdateQuery{CurrentVersion: "0.9.0"})
	require.NoError(t, err)
	require.NotNil(t, out.Release)
	assert.Equal(t, "1.0.0", out.Release.Version)
	require.NotNil(t, out.Artifact)
	assert.Equal(t, "sig", out.Artifact.Signature)
}

func TestCheckUpdate_NoUpdate(t *testing.T) {
	client, err := NewClient(Config{
		BaseURL: "https://cloud.example",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"update_available":false}`), nil
		})},
	})
	require.NoError(t, err)

	_, err = client.CheckUpdate(context.Background(), UpdateQuery{CurrentVersion: "1.0.0"})
	assert.ErrorIs(t, err, ErrNoUpdateAvailable)
}

func TestHandshake_PostsAgentEnvelope(t *testing.T) {
	client, err := NewClient(Config{
		BaseURL: "https://cloud.example",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			assert.Contains(t, string(body), `"fingerprint":"fp"`)
			return jsonResponse(http.StatusOK, `{"device_id":"dev_1","trust_level":"pending","entitlements":{},"telemetry_on":true}`), nil
		})},
	})
	require.NoError(t, err)

	out, err := client.Handshake(context.Background(), HandshakeInput{
		OrganizationID: "org_1",
		Fingerprint:    "fp",
		Name:           "workstation",
		AgentVersion:   "1.0.0",
		Status:         "idle",
		TelemetryOptIn: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "dev_1", out.DeviceID)
	assert.True(t, out.TelemetryOn)
}

func TestClient_MapsAuthAndTransportErrors(t *testing.T) {
	client, err := NewClient(Config{
		BaseURL: "https://cloud.example",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusUnauthorized, `{}`), nil
		})},
	})
	require.NoError(t, err)
	_, err = client.CheckUpdate(context.Background(), UpdateQuery{CurrentVersion: "1.0.0"})
	assert.ErrorIs(t, err, ErrUnauthorized)

	client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("network down")
	})
	_, err = client.CheckUpdate(context.Background(), UpdateQuery{CurrentVersion: "1.0.0"})
	assert.ErrorContains(t, err, "cloud request failed")
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    &http.Request{},
	}
}

func TestNewClient_AppliesDefaultTimeout(t *testing.T) {
	client, err := NewClient(Config{BaseURL: "https://cloud.example"})
	require.NoError(t, err)
	assert.Equal(t, 10*time.Second, client.httpClient.Timeout)
}
