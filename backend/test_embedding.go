package main

import (
	"context"
	"fmt"
	"time"
	"github.com/dika/contextkit/backend/internal/config"
	"github.com/dika/contextkit/backend/internal/service"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}
	client := service.NewOpenRouterServiceClient(cfg)
	start := time.Now()
	_, err = client.GenerateEmbedding(context.Background(), "hello world")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	fmt.Printf("Embedding latency: %v\n", time.Since(start))
}
