package vault

import "time"

// TrustedPluginKey is the vault-layer representation of a plugin signing key.
type TrustedPluginKey struct {
	Fingerprint string
	PublicKey   []byte
	AddedAtUTC  time.Time
}
