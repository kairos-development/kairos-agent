package backup

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/kairos-development/kairos-agent/internal/config"
	"github.com/kairos-development/kairos-agent/internal/vault"
)

// Result contains output paths for a created backup.
type Result struct {
	ArchivePath string
	Checksum    string
}

// Create encrypts a deterministic state archive.
func Create(stateDir string, password string) (Result, error) {
	if strings.TrimSpace(password) == "" {
		return Result{}, fmt.Errorf("backup password is required")
	}
	if info, err := os.Stat(stateDir); err != nil {
		return Result{}, err
	} else if !info.IsDir() {
		return Result{}, fmt.Errorf("state path %s is not a directory", stateDir)
	}
	if _, err := os.Stat(filepath.Join(stateDir, "config.yaml")); err != nil {
		if os.IsNotExist(err) {
			return Result{}, fmt.Errorf("config.yaml is required for backup")
		}
		return Result{}, err
	}

	files := []string{"config.yaml", "trades.sqlite", "vault.db", "audit.log", "journal.log"}
	sort.Strings(files)
	var tarBuffer bytes.Buffer
	tarWriter := tar.NewWriter(&tarBuffer)
	for _, name := range files {
		path := filepath.Join(stateDir, name)
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return Result{}, err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return Result{}, err
		}
		header.Name = name
		header.ModTime = time.Unix(0, 0)
		header.AccessTime = time.Unix(0, 0)
		header.ChangeTime = time.Unix(0, 0)
		if err := tarWriter.WriteHeader(header); err != nil {
			return Result{}, err
		}
		file, err := os.Open(path)
		if err != nil {
			return Result{}, err
		}
		if _, err := io.Copy(tarWriter, file); err != nil {
			file.Close()
			return Result{}, err
		}
		file.Close()
	}
	if err := tarWriter.Close(); err != nil {
		return Result{}, err
	}
	archiveName := fmt.Sprintf("kairos-backup-%s.tar.age", time.Now().UTC().Format("20060102T150405Z"))
	archivePath := filepath.Join(stateDir, archiveName)
	out, err := os.Create(archivePath)
	if err != nil {
		return Result{}, err
	}
	recipient, err := age.NewScryptRecipient(password)
	if err != nil {
		return Result{}, err
	}
	encryptedWriter, err := age.Encrypt(out, recipient)
	if err != nil {
		return Result{}, err
	}
	if _, err := io.Copy(encryptedWriter, bytes.NewReader(tarBuffer.Bytes())); err != nil {
		return Result{}, err
	}
	if err := encryptedWriter.Close(); err != nil {
		return Result{}, err
	}
	if err := out.Close(); err != nil {
		return Result{}, err
	}
	body, err := os.ReadFile(archivePath)
	if err != nil {
		return Result{}, err
	}
	digest := sha256.Sum256(body)
	checksum := hex.EncodeToString(digest[:])
	if err := os.WriteFile(archivePath+".sha256", []byte(checksum+"  "+filepath.Base(archivePath)+"\n"), 0o600); err != nil {
		return Result{}, err
	}
	return Result{ArchivePath: archivePath, Checksum: checksum}, nil
}

// Restore verifies and restores a backup archive into the target state directory.
func Restore(stateDir string, archivePath string, password string) error {
	body, err := os.ReadFile(archivePath)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(body)
	checksumPath := archivePath + ".sha256"
	expectedChecksum, err := os.ReadFile(checksumPath)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(string(expectedChecksum), hex.EncodeToString(digest[:])) {
		return fmt.Errorf("checksum mismatch for %s", archivePath)
	}
	identity, err := age.NewScryptIdentity(password)
	if err != nil {
		return err
	}
	reader, err := age.Decrypt(bytes.NewReader(body), identity)
	if err != nil {
		return err
	}
	tempDir, err := os.MkdirTemp(stateDir, "restore-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		targetPath := filepath.Join(tempDir, header.Name)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		if _, err := io.Copy(file, tarReader); err != nil {
			file.Close()
			return err
		}
		file.Close()
	}
	cfg, err := config.Load(filepath.Join(tempDir, "config.yaml"))
	if err != nil {
		return err
	}
	if cfg.SchemaVersion != config.CurrentSchemaVersion {
		return fmt.Errorf("restore schema_version %d is incompatible with current version %d", cfg.SchemaVersion, config.CurrentSchemaVersion)
	}
	vaultStore, err := vault.OpenOrCreate(filepath.Join(tempDir, "vault.db"), password, "restore-verify")
	if err != nil {
		return err
	}
	vaultStore.Close()
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		source := filepath.Join(tempDir, entry.Name())
		target := filepath.Join(stateDir, entry.Name())
		data, err := os.ReadFile(source)
		if err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o600); err != nil {
			return err
		}
	}
	return nil
}
