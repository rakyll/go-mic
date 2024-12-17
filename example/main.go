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

	done := make(chan struct{})
	go func() {
		time.Sleep(5 * time.Second)
		close(done)
	}()

	buf := mic.NewBuffer()
	if err := stream.Read(buf, done); err != nil {
		log.Fatalf("Error reading from mic: %v", err)
	}

	if err := stream.Close(); err != nil {
		log.Fatalf("Error closing mic: %v", err)
	}
}
