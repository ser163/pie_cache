package pie_cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CacheItem represents an item in the cache
type CacheItem struct {
	Key      string    `json:"key"`      // Cache key
	Data     []byte    `json:"data"`     // Cached data
	ExpireAt time.Time `json:"expireAt"` // Expiration time
	Created  time.Time `json:"created"`  // Creation time
}

// FileCache represents a file-based cache system
type FileCache struct {
	baseDir     string        // Base directory for cache files
	ttl         time.Duration // Default time-to-live for cache items
	dirLevels   int           // Number of directory levels
	prefixLen   int           // Length of directory name prefixes
	purgeOnLoad bool          // Whether to purge expired items on load
}

// NewFileCache creates a new FileCache instance
func NewFileCache(baseDir string, ttl time.Duration) (*FileCache, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %v", err)
	}

	cache := &FileCache{
		baseDir:     baseDir,
		ttl:         ttl,
		dirLevels:   3,    // Three-level directory structure
		prefixLen:   2,    // 2-character prefix for each level
		purgeOnLoad: true, // Purge expired items by default
	}

	return cache, nil
}

// Set adds or updates a cache item with default TTL
func (fc *FileCache) Set(key string, data []byte) error {
	return fc.SetWithTTL(key, data, fc.ttl)
}

// SetWithTTL adds or updates a cache item with specified TTL
func (fc *FileCache) SetWithTTL(key string, data []byte, ttl time.Duration) error {
	expireAt := time.Now().Add(ttl)

	hasKey := strings.ReplaceAll(key, "_info.json", "")
	hasKey = strings.ReplaceAll(hasKey, "_toc.json", "")

	item := CacheItem{
		Key:      hasKey,
		Data:     data,
		ExpireAt: expireAt,
		Created:  time.Now(),
	}

	filePath, err := fc.getFilePath(key)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	jsonData, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal cache item: %v", err)
	}

	if err := ioutil.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %v", err)
	}

	return nil
}

// Get retrieves a cache item
func (fc *FileCache) Get(key string) ([]byte, error) {
	filePath, err := fc.getFilePath(key)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("cache not found")
		}
		return nil, fmt.Errorf("failed to read cache file: %v", err)
	}

	var item CacheItem
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, fmt.Errorf("failed to parse cache file: %v", err)
	}

	if time.Now().After(item.ExpireAt) {
		if fc.purgeOnLoad {
			_ = os.Remove(filePath)
		}
		return nil, errors.New("cache expired")
	}

	return item.Data, nil
}

// GetString retrieves a cache item as string
func (fc *FileCache) GetString(key string) (string, error) {
	data, err := fc.Get(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Exists checks if a cache item exists and is not expired
func (fc *FileCache) Exists(key string) bool {
	filePath, err := fc.getFilePath(key)
	if err != nil {
		return false
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	if fc.purgeOnLoad {
		if _, err := fc.Get(key); err != nil {
			return false
		}
		return true
	}

	return true
}

// Delete removes a cache item
func (fc *FileCache) Delete(key string) error {
	filePath, err := fc.getFilePath(key)
	if err != nil {
		return err
	}

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return errors.New("cache not found")
		}
		return fmt.Errorf("failed to delete cache file: %v", err)
	}

	return nil
}

// PurgeExpired removes all expired cache items
func (fc *FileCache) PurgeExpired() error {
	return filepath.Walk(fc.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".json" {
			return nil
		}

		data, err := ioutil.ReadFile(path)
		if err != nil {
			_ = os.Remove(path)
			return nil
		}

		var item CacheItem
		if err := json.Unmarshal(data, &item); err != nil {
			_ = os.Remove(path)
			return nil
		}

		if time.Now().After(item.ExpireAt) {
			_ = os.Remove(path)
		}

		return nil
	})
}

// ListKeys lists all cache keys (may be slow for large caches)
func (fc *FileCache) ListKeys() ([]string, error) {
	var keys []string

	err := filepath.Walk(fc.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".json" {
			return nil
		}

		relPath, err := filepath.Rel(fc.baseDir, path)
		if err != nil {
			return nil
		}

		relPath = filepath.ToSlash(relPath)
		parts := strings.Split(relPath, "/")
		if len(parts) < fc.dirLevels+1 {
			return nil
		}

		key := parts[fc.dirLevels]
		key = strings.TrimSuffix(key, ".json")

		keys = append(keys, key)

		return nil
	})

	return keys, err
}

// getFilePath generates the file path for a cache key
func (fc *FileCache) getFilePath(key string) (string, error) {
	hasKey := strings.ReplaceAll(key, "_info.json", "")
	hasKey = strings.ReplaceAll(hasKey, "_toc.json", "")
	hash := sha256.Sum256([]byte(hasKey))
	hashStr := hex.EncodeToString(hash[:])

	path := fc.baseDir
	for i := 0; i < fc.dirLevels; i++ {
		start := i * fc.prefixLen
		end := start + fc.prefixLen
		if end > len(hashStr) {
			return "", errors.New("invalid prefix length")
		}
		path = filepath.Join(path, hashStr[start:end])
	}

	return filepath.Join(path, key), nil
}
