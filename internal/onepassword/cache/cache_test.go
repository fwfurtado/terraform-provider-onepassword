package cache

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/1password/onepassword-sdk-go"
)

func TestCacheVaultRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	store, err := NewFileCache(path, time.Minute, time.Minute, time.Minute, nil)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	store.SetVaultID("Vault A", "vault-id")

	reloaded, err := NewFileCache(path, time.Minute, time.Minute, time.Minute, nil)
	if err != nil {
		t.Fatalf("failed to reload cache: %v", err)
	}

	if value, ok := reloaded.GetVaultID("Vault A"); !ok || value != "vault-id" {
		t.Fatalf("expected cached vault id, got ok=%v value=%q", ok, value)
	}
}

func TestCacheSecretEncryptedOnDisk(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	encryptor, err := NewEncryptor("secret-key")
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	store, err := NewFileCache(path, time.Minute, time.Minute, time.Minute, encryptor)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	store.SetSecret("op://vault/item/field", "super-secret")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read cache file: %v", err)
	}

	var file cacheFile
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("failed to decode cache file: %v", err)
	}

	entry, ok := file.Entries[secretKeyPrefix+"op://vault/item/field"]
	if !ok {
		t.Fatalf("expected secret cache entry to be present")
	}

	if !entry.Encrypted {
		t.Fatalf("expected secret entry to be encrypted")
	}

	plaintextEncoded := base64.StdEncoding.EncodeToString([]byte("super-secret"))
	if entry.Value == plaintextEncoded {
		t.Fatalf("expected encrypted payload, got plaintext")
	}
}

func TestCacheItemsTTLExpire(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	store, err := NewFileCache(path, time.Minute, 25*time.Millisecond, time.Minute, nil)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	items := []onepassword.ItemOverview{
		{ID: "1", Title: "Item A"},
	}

	store.SetItemOverviews("vault-id", items)

	time.Sleep(40 * time.Millisecond)

	if cached, ok := store.GetItemOverviews("vault-id"); ok || cached != nil {
		t.Fatalf("expected item cache to expire, got ok=%v items=%v", ok, cached)
	}
}
