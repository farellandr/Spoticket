package main

import (
	"log"

	"github.com/farellandr/spoticket/internal/server"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Fatal("Error loading .env file")
	}

	if err := server.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
