// main.go
package main

import (
	"flag"
	"log"
	"ring/cache"
	"ring/protocol"
)

func main() {
    addr := flag.String("addr", ":8080", "Server address")
    flag.Parse()

    c := cache.NewCache()
    server := protocol.NewServer(*addr, c)

    log.Printf("Starting cache node on %s", *addr)
    if err := server.Start(); err != nil {
        log.Fatal(err)
    }
}