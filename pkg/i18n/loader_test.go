package i18n

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"go.uber.org/zap"
)

// TestLoadCatalogReadsJSON verifies catalog loading from disk.
func TestLoadCatalogReadsJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "translations.json")
	writeFile(t, path, `{"version":1,"locales":{"es":{"hello":"Hola"}}}`)

	catalog, err := LoadCatalog(Config{Path: path, DefaultLocale: "es"}, zap.NewNop())
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	if got := catalog.Default("hello"); got != "Hola" {
		t.Fatalf("expected Hola, got %q", got)
	}
}

// TestLoadCatalogReadsHTTP verifies catalog loading from an HTTP URL.
func TestLoadCatalogReadsHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write([]byte(`{"version":1,"locales":{"es":{"hello":"Hola remota"}}}`))
	}))
	defer server.Close()

	catalog, err := LoadCatalog(Config{Path: server.URL + "/translations.json?token=secret", DefaultLocale: "es"}, zap.NewNop())
	if err != nil {
		t.Fatalf("load remote catalog: %v", err)
	}
	if got := catalog.Default("hello"); got != "Hola remota" {
		t.Fatalf("expected remote translation, got %q", got)
	}
}

// TestLoadCatalogRejectsHTTPFailure verifies remote failures stop startup.
func TestLoadCatalogRejectsHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		http.Error(response, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	if _, err := LoadCatalog(Config{Path: server.URL + "/translations.json"}, zap.NewNop()); err == nil {
		t.Fatal("expected remote status error")
	}
}

// TestLoadCatalogRejectsOversizedHTTPBody verifies remote catalogs are bounded.
func TestLoadCatalogRejectsOversizedHTTPBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Length", strconv.FormatInt(maxCatalogBytes+1, 10))
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if _, err := LoadCatalog(Config{Path: server.URL + "/translations.json"}, zap.NewNop()); err == nil {
		t.Fatal("expected oversized remote catalog error")
	}
}

// TestLoadCatalogAllowsMissingFile verifies missing files fall back to raw keys.
func TestLoadCatalogAllowsMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")

	catalog, err := LoadCatalog(Config{Path: path, DefaultLocale: "es"}, zap.NewNop())
	if err != nil {
		t.Fatalf("load missing catalog: %v", err)
	}

	if got := catalog.Default("hello"); got != "hello" {
		t.Fatalf("expected key fallback, got %q", got)
	}
}

// TestLoadCatalogRejectsInvalidJSON verifies corrupt files fail startup.
func TestLoadCatalogRejectsInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "translations.json")
	writeFile(t, path, `{`)

	if _, err := LoadCatalog(Config{Path: path}, zap.NewNop()); err == nil {
		t.Fatal("expected parse error")
	}
}

// writeFile writes a test file.
func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
