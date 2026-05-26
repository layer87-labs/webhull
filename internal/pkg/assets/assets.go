package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// Service manages cache-busting hashes for static assets.
// On startup it scans the static directory and computes a short SHA-256
// hash for each file. Templates use AssetPath() to append ?v=HASH.
type Service struct {
	mu     sync.RWMutex
	hashes map[string]string // "/static/css/style.css" → "a1b2c3d4"
	logger *zap.Logger
}

// NewService creates an asset service and scans the given directory.
// The urlPrefix is prepended to relative file paths (e.g. "/static").
func NewService(staticDir, urlPrefix string, logger *zap.Logger) *Service {
	s := &Service{
		hashes: make(map[string]string),
		logger: logger,
	}
	s.scan(staticDir, urlPrefix)
	return s
}

// ScanEmbedded hashes files from an embedded FS and registers them under
// urlPrefix. Files are mapped by stripping the fsRoot prefix from their path:
//
//	fsRoot="js", urlPrefix="/static"  →  js/menu.js  becomes  /static/js/menu.js
//
// Entries from the embedded FS are registered with lower priority: a subsequent
// call to scan() (user staticDir) will overwrite any matching key, so user
// files always take precedence over built-in assets.
func (s *Service) ScanEmbedded(embedded fs.FS, fsRoot, urlPrefix string) {
	err := fs.WalkDir(embedded, fsRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".css", ".js", ".webp", ".png", ".jpg", ".jpeg", ".svg", ".woff2", ".woff":
			// hash these
		default:
			return nil
		}

		data, err := fs.ReadFile(embedded, path)
		if err != nil {
			s.logger.Warn("failed to read embedded asset for hashing",
				zap.String("path", path),
				zap.Error(err))
			return nil
		}

		hash := sha256.Sum256(data)
		shortHash := hex.EncodeToString(hash[:])[:8]

		// Strip fsRoot prefix and build URL: /static/js/menu.js
		rel := strings.TrimPrefix(path, fsRoot)
		rel = strings.TrimPrefix(rel, "/")
		urlPath := urlPrefix + "/" + rel

		s.mu.Lock()
		s.hashes[urlPath] = shortHash
		s.mu.Unlock()

		s.logger.Debug("embedded asset hashed",
			zap.String("path", urlPath),
			zap.String("hash", shortHash))

		return nil
	})
	if err != nil {
		s.logger.Error("failed to scan embedded assets", zap.Error(err))
	}
}

// scan walks the static directory and hashes all files.
func (s *Service) scan(dir, urlPrefix string) {
	if dir == "" {
		return
	}

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		// Only hash CSS, JS, and common web assets
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".css", ".js", ".webp", ".png", ".jpg", ".jpeg", ".svg", ".woff2", ".woff":
			// hash these
		default:
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			s.logger.Warn("failed to read asset for hashing",
				zap.String("path", path),
				zap.Error(err))
			return nil
		}

		hash := sha256.Sum256(data)
		shortHash := hex.EncodeToString(hash[:])[:8]

		// Build URL path: /static/css/style.css
		rel, _ := filepath.Rel(dir, path)
		urlPath := urlPrefix + "/" + filepath.ToSlash(rel)

		s.mu.Lock()
		s.hashes[urlPath] = shortHash
		s.mu.Unlock()

		s.logger.Debug("asset hashed",
			zap.String("path", urlPath),
			zap.String("hash", shortHash))

		return nil
	})
	if err != nil {
		s.logger.Error("failed to scan assets directory",
			zap.String("dir", dir),
			zap.Error(err))
	}

	s.logger.Info("assets hashed",
		zap.Int("count", len(s.hashes)))
}

// Path returns the asset URL with cache-busting query parameter.
// e.g. "/static/css/style.css" → "/static/css/style.css?v=a1b2c3d4"
// If the asset is unknown, returns the path unchanged.
func (s *Service) Path(assetPath string) string {
	s.mu.RLock()
	hash, ok := s.hashes[assetPath]
	s.mu.RUnlock()

	if !ok {
		return assetPath
	}
	return fmt.Sprintf("%s?v=%s", assetPath, hash)
}

// Hash returns just the hash for a given asset path, or empty string.
func (s *Service) Hash(assetPath string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hashes[assetPath]
}
