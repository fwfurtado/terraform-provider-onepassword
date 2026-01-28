package cache

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/1password/onepassword-sdk-go"
)

const (
	cacheVersion = 1

	vaultKeyPrefix  = "vault:"
	itemsKeyPrefix  = "items:"
	secretKeyPrefix = "secret:"
)

type cacheFile struct {
	Version int                   `json:"version"`
	Entries map[string]cacheEntry `json:"entries"`
}

type cacheEntry struct {
	Value     string    `json:"value"`
	ExpiresAt time.Time `json:"expires_at"`
	Encrypted bool      `json:"encrypted"`
}

type Cache struct {
	path       string
	entries    map[string]cacheEntry
	ttlVaults  time.Duration
	ttlItems   time.Duration
	ttlSecrets time.Duration
	encryptor  *Encryptor
	mu         sync.Mutex
}

func NewFileCache(path string, ttlVaults, ttlItems, ttlSecrets time.Duration, encryptor *Encryptor) (*Cache, error) {
	c := &Cache{
		path:       path,
		entries:    map[string]cacheEntry{},
		ttlVaults:  ttlVaults,
		ttlItems:   ttlItems,
		ttlSecrets: ttlSecrets,
		encryptor:  encryptor,
	}

	if err := c.load(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Cache) GetVaultID(title string) (string, bool) {
	return c.getString(vaultKeyPrefix + title)
}

func (c *Cache) SetVaultID(title, id string) {
	c.setString(vaultKeyPrefix+title, id, c.ttlVaults, false)
}

func (c *Cache) GetItemOverviews(vaultID string) ([]onepassword.ItemOverview, bool) {
	data, ok := c.getBytes(itemsKeyPrefix + vaultID)
	if !ok {
		return nil, false
	}

	var items []onepassword.ItemOverview
	if err := json.Unmarshal(data, &items); err != nil {
		c.delete(itemsKeyPrefix + vaultID)
		return nil, false
	}

	return items, true
}

func (c *Cache) SetItemOverviews(vaultID string, items []onepassword.ItemOverview) {
	if c.ttlItems <= 0 {
		return
	}
	data, err := json.Marshal(items)
	if err != nil {
		return
	}
	c.setBytes(itemsKeyPrefix+vaultID, data, c.ttlItems, false)
}

func (c *Cache) InvalidateItemOverviews(vaultID string) {
	c.delete(itemsKeyPrefix + vaultID)
}

func (c *Cache) GetSecret(reference string) (string, bool) {
	if c.ttlSecrets <= 0 {
		return "", false
	}
	return c.getString(secretKeyPrefix + reference)
}

func (c *Cache) SetSecret(reference, value string) {
	if c.ttlSecrets <= 0 {
		return
	}
	c.setString(secretKeyPrefix+reference, value, c.ttlSecrets, true)
}

func (c *Cache) getString(key string) (string, bool) {
	data, ok := c.getBytes(key)
	if !ok {
		return "", false
	}
	return string(data), true
}

func (c *Cache) getBytes(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		delete(c.entries, key)
		_ = c.saveLocked()
		return nil, false
	}

	raw, err := base64.StdEncoding.DecodeString(entry.Value)
	if err != nil {
		delete(c.entries, key)
		_ = c.saveLocked()
		return nil, false
	}

	if entry.Encrypted {
		if c.encryptor == nil {
			return nil, false
		}
		plain, err := c.encryptor.Decrypt(raw)
		if err != nil {
			delete(c.entries, key)
			_ = c.saveLocked()
			return nil, false
		}
		return plain, true
	}

	return raw, true
}

func (c *Cache) setString(key, value string, ttl time.Duration, encrypt bool) {
	c.setBytes(key, []byte(value), ttl, encrypt)
}

func (c *Cache) setBytes(key string, value []byte, ttl time.Duration, encrypt bool) {
	if ttl <= 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	payload := value
	encrypted := false
	if encrypt {
		if c.encryptor == nil {
			return
		}
		encoded, err := c.encryptor.Encrypt(value)
		if err != nil {
			return
		}
		payload = encoded
		encrypted = true
	}

	c.entries[key] = cacheEntry{
		Value:     base64.StdEncoding.EncodeToString(payload),
		ExpiresAt: time.Now().Add(ttl),
		Encrypted: encrypted,
	}

	_ = c.saveLocked()
}

func (c *Cache) delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.entries[key]; !ok {
		return
	}

	delete(c.entries, key)
	_ = c.saveLocked()
}

func (c *Cache) load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.path == "" {
		return errors.New("cache path is empty")
	}

	data, err := os.ReadFile(c.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	if len(data) == 0 {
		return nil
	}

	var file cacheFile
	if err := json.Unmarshal(data, &file); err != nil {
		return err
	}

	if file.Version != cacheVersion && file.Version != 0 {
		return errors.New("unsupported cache version")
	}

	c.entries = file.Entries
	c.purgeExpiredLocked()
	_ = c.saveLocked()

	return nil
}

func (c *Cache) purgeExpiredLocked() {
	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
		}
	}
}

func (c *Cache) saveLocked() error {
	if c.path == "" {
		return errors.New("cache path is empty")
	}

	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	payload := cacheFile{
		Version: cacheVersion,
		Entries: c.entries,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	tmpFile := c.path + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0o600); err != nil {
		return err
	}

	return os.Rename(tmpFile, c.path)
}

type Encryptor struct {
	gcm cipher.AEAD
}

func NewEncryptor(key string) (*Encryptor, error) {
	if key == "" {
		return nil, nil
	}
	hash := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(hash[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Encryptor{gcm: gcm}, nil
}

func (e *Encryptor) Encrypt(plain []byte) ([]byte, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := e.gcm.Seal(nil, nonce, plain, nil)
	return append(nonce, ciphertext...), nil
}

func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := e.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce := ciphertext[:nonceSize]
	data := ciphertext[nonceSize:]
	return e.gcm.Open(nil, nonce, data, nil)
}
