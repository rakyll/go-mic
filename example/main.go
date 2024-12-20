package main

import (
	"log"
	"os"
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
	defer stream.Close()

	if err := os.WriteFile("out.wav", stream.EncodedBytes(), 0644); err != nil {
		log.Fatalf("Error writing file: %v", err)
	}
}
