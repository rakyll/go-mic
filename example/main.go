package main

import (
	"log"
	"time"

	"github.com/rakyll/go-mic"
)

func main() {
	stream, err := mic.Open()
	if err != nil {
		log.Fatalf("Error opening mic: %v", err)
	}

	if err := stream.Start(); err != nil {
		log.Fatalf("Error starting mic: %v", err)
	}

	// Record for two seconds...
	time.Sleep(2 * time.Second)
	if err := stream.Stop(); err != nil {
		log.Fatalf("Error stopping mic: %v", err)
	}

	_ = stream.EncodedBytes() // use bytes

	if err := stream.Close(); err != nil {
		log.Fatalf("Error closing mic: %v", err)
	}
}
