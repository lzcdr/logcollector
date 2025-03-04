package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/lzcdr/logcollector/pkg/logcollector"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := godotenv.Load("test/zincsearchintegration/.env"); err != nil {
		log.Println("No .env file found, using default environment variables")
	}

	zincIndexName := os.Getenv("LOGCOLLECTOR_INDEXNAME")
	logFile := os.Getenv("LOGCOLLECTOR_LOGFILE")
	zincURL := os.Getenv("ZINC_URL")
	zincUser := os.Getenv("ZINC_USER")
	zincPass := os.Getenv("ZINC_PASSWORD")
	bufferSize, err := strconv.Atoi(os.Getenv("LOGCOLLECTOR_BUFFERSIZE"))
	if err != nil {
		log.Fatalf("Failed to start log collector: %v", err)
	}
	bufferTimeoutValue, err := strconv.Atoi(os.Getenv("LOGCOLLECTOR_BUFFERTIMEOUT"))
	if err != nil {
		log.Fatalf("Failed to start log collector: %v", err)
	}
	bufferTimeout := time.Duration(bufferTimeoutValue) * time.Second

	logCollector, err := logcollector.NewLogCollector(ctx, zincIndexName, logFile, zincURL, zincUser, zincPass, bufferSize, bufferTimeout)
	if err != nil {
		log.Fatalf("Failed to start log collector: %v", err)
	}

	fmt.Println("Log collector started. Press Ctrl+C to stop.")

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	<-c
	logCollector.Stop()
}
