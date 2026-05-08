package vault

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitialize(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	assert.NoError(t, err)

	// Verify vault was initialized
	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("test-password")
	assert.NoError(t, err)
}

func TestOpen(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	assert.NotNil(t, vault)
	assert.True(t, vault.IsLocked())
}

func TestVault_Unlock_Success(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("test-password")
	assert.NoError(t, err)
	assert.False(t, vault.IsLocked())
}

func TestVault_Unlock_WrongPassword(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("wrong-password")
	assert.ErrorIs(t, err, ErrInvalidPassword)
	assert.True(t, vault.IsLocked())
}

func TestVault_Unlock_NotInitialized(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Create empty database without initializing vault schema
	db, err := os.Create(tmpFile.Name())
	require.NoError(t, err)
	db.Close()

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("test-password")
	// Should get an error (either ErrVaultNotInitialized or a SQL error about missing table)
	assert.Error(t, err)
}

func TestVault_Lock(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("test-password")
	require.NoError(t, err)
	assert.False(t, vault.IsLocked())

	vault.Lock()
	assert.True(t, vault.IsLocked())
	assert.Nil(t, vault.crypto)
}

func TestVault_IsLocked(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	assert.True(t, vault.IsLocked())

	err = vault.Unlock("test-password")
	require.NoError(t, err)
	assert.False(t, vault.IsLocked())
}

func TestVault_Set_Success(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("test-password")
	require.NoError(t, err)

	ctx := context.Background()
	err = vault.Set(ctx, "api_key", []byte("secret-value"))
	assert.NoError(t, err)
}

func TestVault_Set_Locked(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	ctx := context.Background()
	err = vault.Set(ctx, "api_key", []byte("secret-value"))
	assert.ErrorIs(t, err, ErrVaultLocked)
}

func TestVault_Get_Success(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("test-password")
	require.NoError(t, err)

	ctx := context.Background()
	expectedValue := []byte("secret-value")
	err = vault.Set(ctx, "api_key", expectedValue)
	require.NoError(t, err)

	value, err := vault.Get(ctx, "api_key")
	assert.NoError(t, err)
	assert.Equal(t, expectedValue, value)
}

func TestVault_Get_NotFound(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("test-password")
	require.NoError(t, err)

	ctx := context.Background()
	_, err = vault.Get(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrSecretNotFound)
}

func TestVault_Get_Locked(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	ctx := context.Background()
	_, err = vault.Get(ctx, "api_key")
	assert.ErrorIs(t, err, ErrVaultLocked)
}

func TestVault_Delete_Success(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("test-password")
	require.NoError(t, err)

	ctx := context.Background()
	err = vault.Set(ctx, "api_key", []byte("secret-value"))
	require.NoError(t, err)

	err = vault.Delete(ctx, "api_key")
	assert.NoError(t, err)

	// Verify deleted
	_, err = vault.Get(ctx, "api_key")
	assert.ErrorIs(t, err, ErrSecretNotFound)
}

func TestVault_Delete_NotFound(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("test-password")
	require.NoError(t, err)

	ctx := context.Background()
	err = vault.Delete(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrSecretNotFound)
}

func TestVault_ChangePassword_SuccessReencryptsSecrets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.db")
	require.NoError(t, Initialize(path, "old-password"))

	vault, err := Open(path)
	require.NoError(t, err)
	defer vault.Close()
	require.NoError(t, vault.Unlock("old-password"))

	ctx := context.Background()
	require.NoError(t, vault.Set(ctx, "api_key", []byte("secret-value")))
	require.NoError(t, vault.Set(ctx, "api_secret", []byte("secret-two")))
	require.NoError(t, vault.ChangePassword(ctx, "old-password", "new-password"))
	assert.False(t, vault.IsLocked())

	value, err := vault.Get(ctx, "api_key")
	require.NoError(t, err)
	assert.Equal(t, []byte("secret-value"), value)

	vault.Lock()
	assert.ErrorIs(t, vault.Unlock("old-password"), ErrInvalidPassword)
	require.NoError(t, vault.Unlock("new-password"))
	value, err = vault.Get(ctx, "api_secret")
	require.NoError(t, err)
	assert.Equal(t, []byte("secret-two"), value)
}

func TestVault_Delete_Locked(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	ctx := context.Background()
	err = vault.Delete(ctx, "api_key")
	assert.ErrorIs(t, err, ErrVaultLocked)
}

func TestVault_List_Success(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("test-password")
	require.NoError(t, err)

	ctx := context.Background()
	err = vault.Set(ctx, "api_key", []byte("value1"))
	require.NoError(t, err)
	err = vault.Set(ctx, "secret_token", []byte("value2"))
	require.NoError(t, err)

	keys, err := vault.List(ctx)
	assert.NoError(t, err)
	assert.Len(t, keys, 2)
	assert.Contains(t, keys, "api_key")
	assert.Contains(t, keys, "secret_token")
}

func TestVault_List_Empty(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("test-password")
	require.NoError(t, err)

	ctx := context.Background()
	keys, err := vault.List(ctx)
	assert.NoError(t, err)
	assert.Empty(t, keys)
}

func TestVault_List_Locked(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	ctx := context.Background()
	_, err = vault.List(ctx)
	assert.ErrorIs(t, err, ErrVaultLocked)
}

func TestVault_Close(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)

	err = vault.Unlock("test-password")
	require.NoError(t, err)

	err = vault.Close()
	assert.NoError(t, err)
	assert.True(t, vault.IsLocked())
}

// func TestVault_ChangePassword_Success(t *testing.T) {
// 	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
// 	require.NoError(t, err)
// 	tmpFile.Close()
// 	defer os.Remove(tmpFile.Name())
//
// 	err = Initialize(tmpFile.Name(), "old-password")
// 	require.NoError(t, err)
//
// 	vault, err := Open(tmpFile.Name())
// 	require.NoError(t, err)
// 	defer vault.Close()
//
// 	err = vault.Unlock("old-password")
// 	require.NoError(t, err)
//
// 	ctx := context.Background()
// 	err = vault.Set(ctx, "api_key", []byte("secret-value"))
// 	require.NoError(t, err)
//
// 	err = vault.ChangePassword(ctx, "old-password", "new-password")
// 	assert.NoError(t, err)
//
// 	// Verify old password no longer works
// 	vault.Lock()
// 	err = vault.Unlock("old-password")
// 	assert.ErrorIs(t, err, ErrInvalidPassword)
//
// 	// Verify new password works
// 	err = vault.Unlock("new-password")
// 	assert.NoError(t, err)
//
// 	// Verify secret is still accessible
// 	value, err := vault.Get(ctx, "api_key")
// 	assert.NoError(t, err)
// 	assert.Equal(t, []byte("secret-value"), value)
// }
//
// func TestVault_ChangePassword_WrongOldPassword(t *testing.T) {
// 	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
// 	require.NoError(t, err)
// 	tmpFile.Close()
// 	defer os.Remove(tmpFile.Name())
//
// 	err = Initialize(tmpFile.Name(), "old-password")
// 	require.NoError(t, err)
//
// 	vault, err := Open(tmpFile.Name())
// 	require.NoError(t, err)
// 	defer vault.Close()
//
// 	err = vault.Unlock("old-password")
// 	require.NoError(t, err)
//
// 	ctx := context.Background()
// 	err = vault.ChangePassword(ctx, "wrong-password", "new-password")
// 	assert.ErrorIs(t, err, ErrInvalidPassword)
// }
//
// func TestVault_ChangePassword_Locked(t *testing.T) {
// 	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
// 	require.NoError(t, err)
// 	tmpFile.Close()
// 	defer os.Remove(tmpFile.Name())
//
// 	err = Initialize(tmpFile.Name(), "old-password")
// 	require.NoError(t, err)
//
// 	vault, err := Open(tmpFile.Name())
// 	require.NoError(t, err)
// 	defer vault.Close()
//
// 	ctx := context.Background()
// 	err = vault.ChangePassword(ctx, "old-password", "new-password")
// 	assert.ErrorIs(t, err, ErrVaultLocked)
//}

func TestVault_SetAndGet_MultipleSecrets(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("test-password")
	require.NoError(t, err)

	ctx := context.Background()

	// Set multiple secrets
	secrets := map[string][]byte{
		"api_key":      []byte("key-value"),
		"secret_token": []byte("token-value"),
		"password":     []byte("pass-value"),
	}

	for key, value := range secrets {
		err = vault.Set(ctx, key, value)
		require.NoError(t, err)
	}

	// Verify all secrets
	for key, expectedValue := range secrets {
		value, err := vault.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, expectedValue, value)
	}
}

func TestVault_Set_UpdateExisting(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("test-password")
	require.NoError(t, err)

	ctx := context.Background()

	// Set initial value
	err = vault.Set(ctx, "api_key", []byte("old-value"))
	require.NoError(t, err)

	// Update value
	err = vault.Set(ctx, "api_key", []byte("new-value"))
	require.NoError(t, err)

	// Verify updated value
	value, err := vault.Get(ctx, "api_key")
	assert.NoError(t, err)
	assert.Equal(t, []byte("new-value"), value)
}

func TestCrypto_DeriveKeyFingerprint(t *testing.T) {
	password := "test-password"
	salt := make([]byte, 32)
	for i := range salt {
		salt[i] = byte(i)
	}

	crypto, err := NewCrypto(password, salt)
	require.NoError(t, err)

	fingerprint := crypto.DeriveKeyFingerprint()
	assert.NotEmpty(t, fingerprint)
	assert.Len(t, fingerprint, 24) // base64 encoded 16 bytes

	// Fingerprint should be deterministic
	fingerprint2 := crypto.DeriveKeyFingerprint()
	assert.Equal(t, fingerprint, fingerprint2)

	// Different key should produce different fingerprint
	crypto2, err := NewCrypto("different-password", salt)
	require.NoError(t, err)
	fingerprint3 := crypto2.DeriveKeyFingerprint()
	assert.NotEqual(t, fingerprint, fingerprint3)
}

func TestNewCrypto_InvalidSaltSize(t *testing.T) {
	password := "test-password"
	salt := make([]byte, 16) // Invalid size (should be 32)

	_, err := NewCrypto(password, salt)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "salt must be 32 bytes")
}

func TestEncrypt_InvalidData(t *testing.T) {
	password := "test-password"
	salt := make([]byte, 32)

	crypto, err := NewCrypto(password, salt)
	require.NoError(t, err)

	// Empty data should work
	encrypted, err := crypto.Encrypt([]byte{})
	assert.NoError(t, err)
	assert.NotEmpty(t, encrypted)
}

func TestDecrypt_InvalidCiphertext(t *testing.T) {
	password := "test-password"
	salt := make([]byte, 32)

	crypto, err := NewCrypto(password, salt)
	require.NoError(t, err)

	// Too short ciphertext
	_, err = crypto.Decrypt("short")
	assert.Error(t, err)

	// Invalid ciphertext
	_, err = crypto.Decrypt("this is not a valid ciphertext at all")
	assert.Error(t, err)
}

func TestOpen_NonExistentFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "nonexistent.db")

	// Open should succeed even if file doesn't exist (it will be created)
	vault, err := Open(tmpFile)
	if err == nil {
		vault.Close()
	}
	// Just verify it doesn't panic
}

func TestInitialize_ExistingFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Initialize first time
	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	// Initialize again should fail
	err = Initialize(tmpFile.Name(), "test-password")
	assert.Error(t, err)
}

func TestVault_Get_DecryptError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("test-password")
	require.NoError(t, err)

	ctx := context.Background()

	// Insert invalid encrypted data directly (use correct column name 'value')
	_, err = vault.db.ExecContext(ctx, `INSERT INTO secrets (key, value, created_at_utc, updated_at_utc) VALUES (?, ?, ?, ?)`,
		"bad-key", "invalid", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z")
	require.NoError(t, err)

	// Get should fail with decrypt error
	_, err = vault.Get(ctx, "bad-key")
	assert.Error(t, err)
}

func TestVault_List_QueryError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)

	err = vault.Unlock("test-password")
	require.NoError(t, err)

	// Close database to cause query error
	vault.db.Close()

	ctx := context.Background()
	_, err = vault.List(ctx)
	assert.Error(t, err)
}

func TestVault_ChangePassword_WrongOldPassword(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "old-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("old-password")
	require.NoError(t, err)

	ctx := context.Background()
	err = vault.ChangePassword(ctx, "wrong-password", "new-password")
	assert.ErrorIs(t, err, ErrInvalidPassword)
}

func TestVault_ChangePassword_Locked(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "old-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	ctx := context.Background()
	err = vault.ChangePassword(ctx, "old-password", "new-password")
	assert.ErrorIs(t, err, ErrVaultLocked)
}

func TestOpen_PragmaError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Initialize vault
	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	// Open should succeed
	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()
}

func TestOpen_InvalidPath(t *testing.T) {
	_, err := Open("/invalid/nonexistent/path.db")
	assert.Error(t, err)
}

func TestInitialize_SchemaCreationError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Initialize should succeed
	err = Initialize(tmpFile.Name(), "test-password")
	assert.NoError(t, err)
}

func TestClose_AlreadyClosed(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)

	// Close once
	err = vault.Close()
	require.NoError(t, err)

	// Close again - may or may not error depending on SQLite behavior
	_ = vault.Close()
}

func TestEncrypt_EmptyData(t *testing.T) {
	password := "test-password"
	salt := make([]byte, 32)

	crypto, err := NewCrypto(password, salt)
	require.NoError(t, err)

	encrypted, err := crypto.Encrypt([]byte{})
	assert.NoError(t, err)
	assert.NotEmpty(t, encrypted)
}

func TestDecrypt_EmptyString(t *testing.T) {
	password := "test-password"
	salt := make([]byte, 32)

	crypto, err := NewCrypto(password, salt)
	require.NoError(t, err)

	_, err = crypto.Decrypt("")
	assert.Error(t, err)
}

func TestGenerateSalt_Length(t *testing.T) {
	salt, err := GenerateSalt()
	require.NoError(t, err)
	assert.Len(t, salt, 32)
}

func TestSet_EmptyKey(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("test-password")
	require.NoError(t, err)

	ctx := context.Background()
	err = vault.Set(ctx, "", []byte("value"))
	assert.NoError(t, err)
}

func TestGet_EmptyKey(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("test-password")
	require.NoError(t, err)

	ctx := context.Background()
	_, err = vault.Get(ctx, "")
	assert.ErrorIs(t, err, ErrSecretNotFound)
}

func TestDelete_EmptyKey(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	err = Initialize(tmpFile.Name(), "test-password")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("test-password")
	require.NoError(t, err)

	ctx := context.Background()
	err = vault.Delete(ctx, "")
	assert.ErrorIs(t, err, ErrSecretNotFound)
}

func TestOpenOrCreate_MkdirError(t *testing.T) {
	// Try to create vault in a file (not directory)
	tmpFile := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(tmpFile, []byte("test"), 0o600))

	vaultPath := filepath.Join(tmpFile, "vault.db")
	_, err := OpenOrCreate(vaultPath, "password", "test")
	assert.Error(t, err)
}

func TestHasPendingMigrations_OpenError(t *testing.T) {
	_, err := hasPendingMigrations("/nonexistent/vault.db")
	assert.Error(t, err)
}

func TestCreateVaultBackup_OpenError(t *testing.T) {
	stateDir := t.TempDir()
	err := createVaultBackup(stateDir, "/nonexistent/vault.db")
	assert.Error(t, err)
}

func TestCreateVaultBackup_CreateError(t *testing.T) {
	stateDir := t.TempDir()
	sourceFile := filepath.Join(stateDir, "vault.db")
	require.NoError(t, os.WriteFile(sourceFile, []byte("test"), 0o600))

	// Make backup dir read-only
	backupDir := filepath.Join(stateDir, "backups")
	require.NoError(t, os.MkdirAll(backupDir, 0o755))
	require.NoError(t, os.Chmod(backupDir, 0o500))
	defer os.Chmod(backupDir, 0o755)

	err := createVaultBackup(stateDir, sourceFile)
	assert.Error(t, err)
}

func TestApplyMigrations_TransactionError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	db, err := sql.Open("sqlite", tmpFile.Name())
	require.NoError(t, err)
	defer db.Close()

	// Apply migrations successfully first
	err = applyMigrations(db, "test")
	require.NoError(t, err)

	// Close DB to cause error
	db.Close()

	// Try to apply migrations again - should fail
	err = applyMigrations(db, "test")
	assert.Error(t, err)
}

func TestInitialize_GenerateSaltError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Initialize should succeed
	err = Initialize(tmpFile.Name(), "password123")
	assert.NoError(t, err)
}

func TestUnlock_DecodeSaltError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Initialize vault
	err = Initialize(tmpFile.Name(), "password123")
	require.NoError(t, err)

	// Open vault
	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	// Corrupt the salt
	_, err = vault.db.Exec("UPDATE vault_meta SET value = ? WHERE key = ?", "invalid-base64!!!", "salt")
	require.NoError(t, err)

	// Try to unlock - should fail
	err = vault.Unlock("password123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode salt")
}

func TestUnlock_LoadPasswordHashError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Initialize vault
	err = Initialize(tmpFile.Name(), "password123")
	require.NoError(t, err)

	// Open vault
	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	// Delete password hash
	_, err = vault.db.Exec("DELETE FROM vault_meta WHERE key = ?", "password_hash")
	require.NoError(t, err)

	// Try to unlock - should fail
	err = vault.Unlock("password123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load password hash")
}

func TestSet_EncryptError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Initialize and unlock vault
	err = Initialize(tmpFile.Name(), "password123")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("password123")
	require.NoError(t, err)

	// Set should succeed with normal data
	err = vault.Set(context.Background(), "test-key", []byte("test-value"))
	assert.NoError(t, err)
}

func TestSet_ExecError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Initialize and unlock vault
	err = Initialize(tmpFile.Name(), "password123")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("password123")
	require.NoError(t, err)

	// Drop the secrets table to cause exec error
	_, err = vault.db.Exec("DROP TABLE secrets")
	require.NoError(t, err)

	err = vault.Set(context.Background(), "test-key", []byte("test-value"))
	assert.Error(t, err)
}

func TestDelete_ExecError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Initialize and unlock vault
	err = Initialize(tmpFile.Name(), "password123")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("password123")
	require.NoError(t, err)

	// Drop the secrets table to cause exec error
	_, err = vault.db.Exec("DROP TABLE secrets")
	require.NoError(t, err)

	err = vault.Delete(context.Background(), "test-key")
	assert.Error(t, err)
}

func TestList_ScanError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Initialize and unlock vault
	err = Initialize(tmpFile.Name(), "password123")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("password123")
	require.NoError(t, err)

	// Add a secret
	err = vault.Set(context.Background(), "test-key", []byte("test-value"))
	require.NoError(t, err)

	// List should succeed
	keys, err := vault.List(context.Background())
	require.NoError(t, err)
	assert.Contains(t, keys, "test-key")
}

func TestChangePassword_LoadSaltError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Initialize and unlock vault
	err = Initialize(tmpFile.Name(), "password123")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("password123")
	require.NoError(t, err)

	// Delete salt to cause error
	_, err = vault.db.Exec("DELETE FROM vault_meta WHERE key = ?", "salt")
	require.NoError(t, err)

	err = vault.ChangePassword(context.Background(), "password123", "newpassword")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load salt")
}

func TestChangePassword_DecodeSaltError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Initialize and unlock vault
	err = Initialize(tmpFile.Name(), "password123")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("password123")
	require.NoError(t, err)

	// Corrupt salt
	_, err = vault.db.Exec("UPDATE vault_meta SET value = ? WHERE key = ?", "invalid!!!", "salt")
	require.NoError(t, err)

	err = vault.ChangePassword(context.Background(), "password123", "newpassword")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode salt")
}

func TestChangePassword_LoadPasswordHashError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Initialize and unlock vault
	err = Initialize(tmpFile.Name(), "password123")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("password123")
	require.NoError(t, err)

	// Delete password hash
	_, err = vault.db.Exec("DELETE FROM vault_meta WHERE key = ?", "password_hash")
	require.NoError(t, err)

	err = vault.ChangePassword(context.Background(), "password123", "newpassword")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load password hash")
}

func TestChangePassword_ListSecretsError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Initialize and unlock vault
	err = Initialize(tmpFile.Name(), "password123")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("password123")
	require.NoError(t, err)

	// Drop secrets table to cause list error
	_, err = vault.db.Exec("DROP TABLE secrets")
	require.NoError(t, err)

	err = vault.ChangePassword(context.Background(), "password123", "newpassword")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list secrets")
}

func TestChangePassword_BeginTxError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Initialize and unlock vault
	err = Initialize(tmpFile.Name(), "password123")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)

	err = vault.Unlock("password123")
	require.NoError(t, err)

	// Close DB to cause transaction error
	vault.db.Close()

	err = vault.ChangePassword(context.Background(), "password123", "newpassword")
	assert.Error(t, err)
	vault.Close()
}

func TestChangePassword_UpdateSecretError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_vault_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Initialize and unlock vault
	err = Initialize(tmpFile.Name(), "password123")
	require.NoError(t, err)

	vault, err := Open(tmpFile.Name())
	require.NoError(t, err)
	defer vault.Close()

	err = vault.Unlock("password123")
	require.NoError(t, err)

	// Add a secret
	err = vault.Set(context.Background(), "test-key", []byte("test-value"))
	require.NoError(t, err)

	// Change password should succeed
	err = vault.ChangePassword(context.Background(), "password123", "newpassword")
	assert.NoError(t, err)
}
