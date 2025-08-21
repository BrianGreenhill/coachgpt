package main

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gregjones/httpcache"
)

// Simple test to verify httpcache is working
func testCache() {
	fmt.Println("Testing httpcache functionality...")

	// Create a transport with memory cache
	transport := httpcache.NewMemoryCacheTransport()
	transport.MarkCachedResponses = true // Add X-From-Cache header
	client := &http.Client{Transport: transport}

	// Make a request to a public API
	url := "https://httpbin.org/get"

	fmt.Printf("Making first request to %s...\n", url)
	start1 := time.Now()
	resp1, err := client.Get(url)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	duration1 := time.Since(start1)

	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	fmt.Printf("First request took: %v\n", duration1)
	fmt.Printf("X-From-Cache header: %s\n", resp1.Header.Get("X-From-Cache"))
	fmt.Printf("Response length: %d bytes\n", len(body1))

	// Make the same request again - should be cached
	fmt.Printf("\nMaking second request to %s...\n", url)
	start2 := time.Now()
	resp2, err := client.Get(url)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	duration2 := time.Since(start2)

	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	fmt.Printf("Second request took: %v\n", duration2)
	fmt.Printf("X-From-Cache header: %s\n", resp2.Header.Get("X-From-Cache"))
	fmt.Printf("Response length: %d bytes\n", len(body2))

	// Verify caching worked
	if resp2.Header.Get("X-From-Cache") == "1" {
		fmt.Printf("\n✅ SUCCESS: Second request was served from cache!\n")
		fmt.Printf("Speed improvement: %.2fx faster\n", float64(duration1)/float64(duration2))
	} else {
		fmt.Printf("\n❌ WARNING: Second request was not cached\n")
	}
}

func init() {
	// Uncomment this line to run the cache test
	// testCache()
}
