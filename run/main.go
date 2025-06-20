package main

import (
	"fmt"
	"github.com/ser163/pie_cache"
	"time"
)

func main() {
	// Create cache with default TTL of 5 minutes
	cache, err := pie_cache.NewFileCache("/tmp/my_cache", 5*time.Minute)
	if err != nil {
		panic(err)
	}

	// Set a value
	err = cache.Set("user:123", []byte("user data"))
	if err != nil {
		panic(err)
	}

	// Get a value
	data, err := cache.Get("user:123")
	if err != nil {
		panic(err)
	}
	fmt.Println("Got:", string(data))

	// Check if key exists
	if cache.Exists("user:123") {
		fmt.Println("Key exists")
	}

	// Delete a key
	err = cache.Delete("user:123")
	if err != nil {
		panic(err)
	}
}
