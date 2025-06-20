package pie_cache

import (
	"os"
	"testing"
	"time"
)

func TestFileCache(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "pie_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create cache with 1 second TTL
	cache, err := NewFileCache(tempDir, time.Second)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Test Set and Get
	key := "test_key"
	value := []byte("test_value")

	if err := cache.Set(key, value); err != nil {
		t.Errorf("Set failed: %v", err)
	}

	gotValue, err := cache.Get(key)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}

	if string(gotValue) != string(value) {
		t.Errorf("Expected %q, got %q", value, gotValue)
	}

	// Test Exists
	if !cache.Exists(key) {
		t.Error("Exists returned false for existing key")
	}

	// Test expiration
	time.Sleep(2 * time.Second)
	if _, err := cache.Get(key); err == nil {
		t.Error("Expected error for expired item")
	}

	// Test Delete
	if err := cache.Set(key, value); err != nil {
		t.Errorf("Set failed: %v", err)
	}

	if err := cache.Delete(key); err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	if cache.Exists(key) {
		t.Error("Exists returned true for deleted key")
	}

	// Test PurgeExpired
	if err := cache.SetWithTTL("expired1", []byte("data1"), time.Millisecond); err != nil {
		t.Errorf("SetWithTTL failed: %v", err)
	}
	if err := cache.SetWithTTL("valid1", []byte("data2"), time.Minute); err != nil {
		t.Errorf("SetWithTTL failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	if err := cache.PurgeExpired(); err != nil {
		t.Errorf("PurgeExpired failed: %v", err)
	}

	if cache.Exists("expired1") {
		t.Error("Expired item not purged")
	}
	if !cache.Exists("valid1") {
		t.Error("Valid item was purged")
	}
}
