// test_client.go
// test_client.go
package main

import (
	"fmt"
	"log"
	"ring/client"
	"ring/cluster"
	"time"
)

func main() {
    // Initialize client with all cache nodes
    c := client.NewClient([]string{
        "localhost:8080",
        "localhost:8081",
        "localhost:8082",
    })

    fmt.Println("=== Testing Distributed Cache ===")

    // Test 0: Check key distribution
    fmt.Println("Test 0: Checking key distribution across nodes")
    checkDistribution()
    fmt.Println()

    // Test 1: Set and Get
    fmt.Println("Test 1: Setting key 'user:1'")
    err := c.Set("user:1", "John Doe", 300) // 300 second TTL
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("✓ Set successful")

    fmt.Println("Getting key 'user:1'")
    value, err := c.Get("user:1")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("✓ Got value: %s\n\n", value)

    // Test 2: Multiple keys (distributed across nodes)
    fmt.Println("Test 2: Setting multiple keys")
    keys := map[string]string{
        "product:100": "Laptop",
        "product:101": "Mouse",
        "product:102": "Keyboard",
        "session:abc": "active",
        "session:xyz": "expired",
    }

    for key, val := range keys {
        err := c.Set(key, val, 600)
        if err != nil {
            log.Printf("Error setting %s: %v", key, err)
        } else {
            fmt.Printf("✓ Set %s = %s\n", key, val)
        }
    }

    fmt.Println("\nRetrieving all keys:")
    for key := range keys {
        val, err := c.Get(key)
        if err != nil {
            log.Printf("Error getting %s: %v", key, err)
        } else {
            fmt.Printf("✓ Got %s = %s\n", key, val)
        }
    }

    // Test 3: TTL expiration
    fmt.Println("\nTest 3: Testing TTL expiration")
    c.Set("temp:key", "temporary value", 3) // 3 second TTL
    val, _ := c.Get("temp:key")
    fmt.Printf("✓ Immediately after set: %s\n", val)
    
    fmt.Println("Waiting 4 seconds for expiration...")
    time.Sleep(4 * time.Second)
    
    _, err = c.Get("temp:key")
    if err != nil {
        fmt.Println("✓ Key expired as expected")
    } else {
        fmt.Println("✗ Key should have expired!")
    }

    // Test 4: Delete
    fmt.Println("\nTest 4: Testing delete")
    c.Set("delete:me", "will be deleted", 0)
    fmt.Println("✓ Set delete:me")
    
    err = c.Delete("delete:me")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("✓ Deleted delete:me")
    
    _, err = c.Get("delete:me")
    if err != nil {
        fmt.Println("✓ Key not found after delete (as expected)")
    }

    fmt.Println("\n=== All tests completed ===")
}

func checkDistribution() {
    cl := cluster.NewCluster()
    cl.AddNode("localhost:8080")
    cl.AddNode("localhost:8081")
    cl.AddNode("localhost:8082")

    testKeys := []string{
        "user:1", "user:2", "user:3",
        "product:100", "product:101", "product:102",
        "session:abc", "session:xyz",
        "order:999", "order:1000",
    }
    
    fmt.Println("Key distribution across nodes:")
    fmt.Println("------------------------------")
    
    // Count keys per node
    nodeCount := make(map[string]int)
    
    for _, key := range testKeys {
        node := cl.GetNodeForKey(key)
        nodeCount[node]++
        fmt.Printf("  %s -> %s\n", key, node)
    }
    
    fmt.Println("\nDistribution summary:")
    for node, count := range nodeCount {
        fmt.Printf("  %s: %d keys\n", node, count)
    }
}