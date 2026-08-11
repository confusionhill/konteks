package config

import (
	"encoding/json"
	"os"
	"strconv"
)

type Config struct {
	Port                     int    `json:"port"`
	DatabaseURL              string `json:"database_url"`
	OpenRouterAPIKey         string `json:"openrouter_api_key"`
	OpenRouterBaseURL        string `json:"openrouter_base_url"`
	OpenRouterEmbeddingModel string `json:"openrouter_embedding_model"`
	OpenRouterChatModel      string `json:"openrouter_chat_model"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	// Environment variable overrides
	if envPort := os.Getenv("PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			cfg.Port = p
		}
	}
	if envDB := os.Getenv("DATABASE_URL"); envDB != "" {
		cfg.DatabaseURL = envDB
	}
	if envAPIKey := os.Getenv("OPENROUTER_API_KEY"); envAPIKey != "" {
		cfg.OpenRouterAPIKey = envAPIKey
	}
	if envEmbeddingModel := os.Getenv("OPENROUTER_EMBEDDING_MODEL"); envEmbeddingModel != "" {
		cfg.OpenRouterEmbeddingModel = envEmbeddingModel
	}
	if envChatModel := os.Getenv("OPENROUTER_CHAT_MODEL"); envChatModel != "" {
		cfg.OpenRouterChatModel = envChatModel
	}

	return &cfg, nil
}
