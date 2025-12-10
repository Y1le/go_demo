package main

import (
	"liam/internal/app"
	"log"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatalf("Failed to run application: %v", err)
	}
}
