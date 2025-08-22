package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mycelian/mycelian-memory/client"
)

func main() {
	// Test dev mode authentication
	c, err := client.NewWithDevMode("http://localhost:11545")
	if err != nil {
		log.Fatalf("NewWithDevMode failed: %v", err)
	}
	defer c.Close()

	fmt.Println("Testing authentication...")

	// Try an async operation that should trigger error logging
	_, err = c.AddEntry(context.Background(), "3ea7a8b3-93b4-44d1-b18e-f0a5b76ae31c", "2be61b26-e6f1-469d-b65d-ee4c5d5ee485", client.AddEntryRequest{
		RawEntry: "Test async auth",
		Summary: "Testing",
	})
	if err != nil {
		log.Printf("AddEntry returned error: %v", err)
	} else {
		fmt.Println("AddEntry enqueued successfully")
	}

	// Wait for consistency - this should trigger the HTTP call and error logging
	fmt.Println("Awaiting consistency...")
	err = c.AwaitConsistency(context.Background(), "2be61b26-e6f1-469d-b65d-ee4c5d5ee485")
	if err != nil {
		log.Printf("AwaitConsistency failed: %v", err)
	} else {
		fmt.Println("Consistency achieved")
	}
}
