package onepasswordprovider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zalando/go-keyring"
)

const (
	cacheKeyEnvVar        = "OP_CACHE_KEY"
	cacheKeyringService   = "onepassword-tf-provider"
	cacheKeyringUser      = "cache-key"
	defaultCacheEnabled   = true
	defaultCacheVaultTTL  = 30 * time.Minute
	defaultCacheItemTTL   = 10 * time.Minute
	defaultCacheSecretTTL = 0 * time.Second
	defaultCacheFileName  = "cache.json"
)

func defaultCachePath() string {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "onepassword-tf-provider", defaultCacheFileName)
}

func resolveCacheKey(useKeyring bool) (string, error) {
	if useKeyring {
		key, err := keyring.Get(cacheKeyringService, cacheKeyringUser)
		if err == nil && key != "" {
			return key, nil
		}
	}

	key := strings.TrimSpace(os.Getenv(cacheKeyEnvVar))
	if key != "" {
		return key, nil
	}

	if useKeyring {
		return "", fmt.Errorf("keyring entry %q/%q not found and %s is not set", cacheKeyringService, cacheKeyringUser, cacheKeyEnvVar)
	}

	return "", fmt.Errorf("%s is not set", cacheKeyEnvVar)
}
