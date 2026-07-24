package i18n

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	// catalogHTTPTimeout bounds remote catalog startup reads.
	catalogHTTPTimeout = 10 * time.Second
	// maxCatalogBytes bounds local and remote translation catalog sizes.
	maxCatalogBytes int64 = 8 << 20
)

// document stores the JSON translation file shape.
type document struct {
	// Version stores the catalog file version.
	Version int `json:"version"`
	// Locales stores translations by locale and key.
	Locales map[string]map[string]string `json:"locales"`
}

// LoadCatalog reads the configured translation catalog.
func LoadCatalog(config Config, log *zap.Logger) (*Catalog, error) {
	config = config.Normalize()
	data, missing, err := readCatalog(config.Path)
	if err != nil {
		return nil, err
	}
	source := catalogSourceLabel(config.Path)
	if missing {
		if log != nil {
			log.Warn("i18n catalog missing", zap.String("source", source))
		}

		return NewCatalog(config, nil), nil
	}

	entries, err := parseCatalog(data)
	if err != nil {
		return nil, fmt.Errorf("parse i18n catalog: %w", err)
	}

	if log != nil {
		log.Info("i18n catalog loaded", zap.String("source", source), zap.Int("locales", len(entries)), zap.Int("keys", countEntries(entries)))
	}

	return NewCatalog(config, entries), nil
}

// readCatalog reads a bounded local file or downloads a bounded HTTP catalog.
func readCatalog(source string) ([]byte, bool, error) {
	if isHTTPSource(source) {
		data, err := fetchCatalog(source)
		if err != nil {
			return nil, false, err
		}

		return data, false, nil
	}
	file, err := os.Open(source)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, true, nil
		}

		return nil, false, fmt.Errorf("open i18n catalog: %w", err)
	}
	defer file.Close()
	data, err := readBounded(file)
	if err != nil {
		return nil, false, fmt.Errorf("read i18n catalog: %w", err)
	}

	return data, false, nil
}

// fetchCatalog downloads one HTTP catalog during startup.
func fetchCatalog(source string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), catalogHTTPTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return nil, fmt.Errorf("create i18n catalog request: %w", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch i18n catalog from %s: %w", catalogSourceLabel(source), err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("fetch i18n catalog from %s: unexpected HTTP status %d", catalogSourceLabel(source), response.StatusCode)
	}
	if response.ContentLength > maxCatalogBytes {
		return nil, fmt.Errorf("fetch i18n catalog from %s: catalog exceeds %d bytes", catalogSourceLabel(source), maxCatalogBytes)
	}
	data, err := readBounded(response.Body)
	if err != nil {
		return nil, fmt.Errorf("fetch i18n catalog from %s: %w", catalogSourceLabel(source), err)
	}

	return data, nil
}

// readBounded reads one catalog while enforcing the shared size limit.
func readBounded(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maxCatalogBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxCatalogBytes {
		return nil, fmt.Errorf("catalog exceeds %d bytes", maxCatalogBytes)
	}

	return data, nil
}

// isHTTPSource reports whether source uses a supported remote scheme.
func isHTTPSource(source string) bool {
	lower := strings.ToLower(source)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

// catalogSourceLabel removes URL credentials and query parameters from logs.
func catalogSourceLabel(source string) string {
	if !isHTTPSource(source) {
		return source
	}
	parsed, err := url.Parse(source)
	if err != nil {
		return "remote"
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""

	return parsed.String()
}

// NewTranslator exposes a catalog as translator.
func NewTranslator(catalog *Catalog) Translator {
	return catalog
}

// parseCatalog decodes catalog JSON.
func parseCatalog(data []byte) (map[Locale]map[Key]string, error) {
	var raw document
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	entries := make(map[Locale]map[Key]string, len(raw.Locales))
	for locale, values := range raw.Locales {
		keyed := make(map[Key]string, len(values))
		for key, value := range values {
			keyed[Key(key)] = value
		}
		entries[Locale(locale)] = keyed
	}

	return entries, nil
}

// countEntries counts translation keys.
func countEntries(entries map[Locale]map[Key]string) int {
	total := 0
	for _, values := range entries {
		total += len(values)
	}

	return total
}
