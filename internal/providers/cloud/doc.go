// Package cloud contains the HTTP adapter for optional Kairos Cloud APIs.
//
// The adapter is intentionally thin: it performs transport, JSON encoding,
// and status mapping only. Local runtime safety, risk decisions, and license
// fallback behavior remain owned by the agent service layer.
package cloud
