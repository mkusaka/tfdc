package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const schemaVersion = "v1"

type Store struct {
	dir     string
	ttl     time.Duration
	enabled bool
	now     func() time.Time
}

type entry struct {
	Schema      string `json:"schema"`
	KeyHash     string `json:"key_hash"`
	Method      string `json:"method"`
	URL         string `json:"url"`
	CreatedAt   string `json:"created_at"`
	ExpiresAt   string `json:"expires_at"`
	Status      int    `json:"status"`
	ContentType string `json:"content_type,omitempty"`
	Body        []byte `json:"body"`
}

type meta struct {
	SchemaVersion string `json:"schema_version"`
}

func NewStore(dir string, ttl time.Duration, enabled bool) (*Store, error) {
	s := &Store{
		dir:     dir,
		ttl:     ttl,
		enabled: enabled,
		now:     time.Now,
	}
	if !enabled {
		return s, nil
	}

	if ttl <= 0 {
		return nil, fmt.Errorf("cache ttl must be positive")
	}

	if err := os.MkdirAll(filepath.Join(dir, schemaVersion, "entries"), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(dir, schemaVersion, "tmp"), 0o755); err != nil {
		return nil, err
	}

	metaPath := filepath.Join(dir, schemaVersion, "meta.json")
	b, err := json.MarshalIndent(meta{SchemaVersion: schemaVersion}, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(metaPath, b, 0o644); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Store) Get(method, rawURL string) ([]byte, bool, error) {
	if !s.enabled {
		return nil, false, nil
	}
	path, keyHash := s.entryPath(method, rawURL)

	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, err
	}

	var e entry
	if err := json.Unmarshal(b, &e); err != nil {
		_ = os.Remove(path)
		return nil, false, nil
	}

	if e.Schema != schemaVersion || e.KeyHash != keyHash {
		_ = os.Remove(path)
		return nil, false, nil
	}

	expiresAt, err := time.Parse(time.RFC3339Nano, e.ExpiresAt)
	if err != nil {
		_ = os.Remove(path)
		return nil, false, nil
	}

	if s.now().After(expiresAt) {
		_ = os.Remove(path)
		return nil, false, nil
	}

	return e.Body, true, nil
}

func (s *Store) Set(method, rawURL string, status int, contentType string, body []byte) error {
	if !s.enabled {
		return nil
	}
	entryPath, keyHash := s.entryPath(method, rawURL)
	if err := os.MkdirAll(filepath.Dir(entryPath), 0o755); err != nil {
		return err
	}

	now := s.now().UTC()
	e := entry{
		Schema:      schemaVersion,
		KeyHash:     keyHash,
		Method:      strings.ToUpper(method),
		URL:         rawURL,
		CreatedAt:   now.Format(time.RFC3339Nano),
		ExpiresAt:   now.Add(s.ttl).Format(time.RFC3339Nano),
		Status:      status,
		ContentType: contentType,
		Body:        body,
	}

	b, err := json.Marshal(e)
	if err != nil {
		return err
	}

	tmpPath := filepath.Join(s.dir, schemaVersion, "tmp", fmt.Sprintf("%s.tmp", keyHash))
	if err := os.WriteFile(tmpPath, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, entryPath)
}

func (s *Store) entryPath(method, rawURL string) (string, string) {
	h := sha256.Sum256([]byte(strings.ToUpper(method) + " " + rawURL))
	keyHash := hex.EncodeToString(h[:])
	prefix := keyHash[:2]
	return filepath.Join(s.dir, schemaVersion, "entries", prefix, keyHash+".json"), keyHash
}
