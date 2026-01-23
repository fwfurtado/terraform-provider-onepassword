package client

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadDotEnv(t *testing.T) {
	t.Helper()

	root := findRepoRoot(t)
	envPath := filepath.Join(root, ".env")

	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		template := strings.Join([]string{
			"# 1Password provider test configuration",
			"# Fill these values to run integration tests locally.",
			"",
			"OP_SERVICE_ACCOUNT_TOKEN=",
			"OP_ACCOUNT_NAME=",
			"",
			"OP_TEST_VAULT=",
			"OP_TEST_ITEM=",
			"OP_TEST_FIELD=",
			"OP_TEST_SECTION=",
			"",
		}, "\n")

		if writeErr := os.WriteFile(envPath, []byte(template), 0644); writeErr != nil {
			t.Fatalf("failed to create .env template: %v", writeErr)
		}
	}

	file, err := os.Open(envPath)
	if err != nil {
		t.Fatalf("failed to open .env: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)

		if key == "" || value == "" {
			continue
		}

		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("failed reading .env: %v", err)
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("failed to locate repo root (go.mod)")
		}
		dir = parent
	}
}

func newTestClient(t *testing.T) *ClientWrapper {
	t.Helper()

	loadDotEnv(t)

	if token := os.Getenv("OP_SERVICE_ACCOUNT_TOKEN"); token != "" {
		client, err := NewServiceAccount(context.Background(), "test", token)
		if err != nil {
			t.Fatalf("failed to create service account client: %v", err)
		}
		return client
	}

	if account := os.Getenv("OP_ACCOUNT_NAME"); account != "" {
		client, err := NewDesktopAppIntegration(context.Background(), "test", account)
		if err != nil {
			t.Fatalf("failed to create desktop app client: %v", err)
		}
		return client
	}

	t.Skip("set OP_SERVICE_ACCOUNT_TOKEN or OP_ACCOUNT_NAME in .env to run integration tests")
	return nil
}

func getEnvOrSkip(t *testing.T, key string) string {
	t.Helper()

	value := os.Getenv(key)
	if value == "" {
		t.Skipf("missing %s in .env", key)
	}

	return value
}
