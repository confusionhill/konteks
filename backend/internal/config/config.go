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
	OpenRouterEmbeddingModel  string  `json:"openrouter_embedding_model"`
	OpenRouterReasoningModel  string  `json:"openrouter_reasoning_model"`
	OpenRouterEvaluationModel  string  `json:"openrouter_evaluation_model"`
	S3Endpoint               string  `json:"s3_endpoint"`
	S3AccessKey              string  `json:"s3_access_key"`
	S3SecretKey              string  `json:"s3_secret_key"`
	S3Bucket                 string  `json:"s3_bucket"`
	EvaluationSampleRate     float64 `json:"evaluation_sample_rate"`
	JWTSecret                string  `json:"jwt_secret"`
	EncryptionMasterKey      string  `json:"encryption_master_key"`
	VectorProvider           string  `json:"vector_provider"`
	CloudflareAccountID      string  `json:"cloudflare_account_id"`
	CloudflareAPIToken       string  `json:"cloudflare_api_token"`
	CloudflareMemoryIndex    string  `json:"cloudflare_memory_index"`
	CloudflareArtifactIndex  string  `json:"cloudflare_artifact_index"`
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
	if envBaseURL := os.Getenv("OPENROUTER_BASE_URL"); envBaseURL != "" {
		cfg.OpenRouterBaseURL = envBaseURL
	}
	if envEmbeddingModel := os.Getenv("OPENROUTER_EMBEDDING_MODEL"); envEmbeddingModel != "" {
		cfg.OpenRouterEmbeddingModel = envEmbeddingModel
	}
	if envReasoningModel := os.Getenv("OPENROUTER_REASONING_MODEL"); envReasoningModel != "" {
		cfg.OpenRouterReasoningModel = envReasoningModel
	}
	if envEvaluationModel := os.Getenv("OPENROUTER_EVALUATION_MODEL"); envEvaluationModel != "" {
		cfg.OpenRouterEvaluationModel = envEvaluationModel
	}
	if envS3Endpoint := os.Getenv("AWS_S3_ENDPOINT"); envS3Endpoint != "" {
		cfg.S3Endpoint = envS3Endpoint
	}
	if envS3Access := os.Getenv("AWS_ACCESS_KEY_ID"); envS3Access != "" {
		cfg.S3AccessKey = envS3Access
	}
	if envS3Secret := os.Getenv("AWS_SECRET_ACCESS_KEY"); envS3Secret != "" {
		cfg.S3SecretKey = envS3Secret
	}
	if envS3Bucket := os.Getenv("AWS_S3_BUCKET"); envS3Bucket != "" {
		cfg.S3Bucket = envS3Bucket
	}
	if envEvalRate := os.Getenv("EVALUATION_SAMPLE_RATE"); envEvalRate != "" {
		if rate, err := strconv.ParseFloat(envEvalRate, 64); err == nil {
			cfg.EvaluationSampleRate = rate
		}
	}
	if envJWT := os.Getenv("JWT_SECRET"); envJWT != "" {
		cfg.JWTSecret = envJWT
	}
	if envEncKey := os.Getenv("ENCRYPTION_MASTER_KEY"); envEncKey != "" {
		cfg.EncryptionMasterKey = envEncKey
	}

	// Default fallbacks
	if cfg.OpenRouterBaseURL == "" {
		cfg.OpenRouterBaseURL = "https://openrouter.ai/api/v1"
	}
	if cfg.OpenRouterReasoningModel == "" {
		cfg.OpenRouterReasoningModel = "deepseek/deepseek-chat"
	}
	if cfg.OpenRouterEvaluationModel == "" {
		cfg.OpenRouterEvaluationModel = "nvidia/nemotron-3.5-lightning:nitro"
	}
	if cfg.OpenRouterEmbeddingModel == "" {
		cfg.OpenRouterEmbeddingModel = "qwen/qwen3-embedding-8b"
	}
	if cfg.S3Bucket == "" {
		cfg.S3Bucket = "contextkit-archive"
	}
	if cfg.EvaluationSampleRate == 0 {
		cfg.EvaluationSampleRate = 1.0 // Temporary default for testing
	}
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "contextkit_super_secret_jwt_key_2026_change_in_prod"
	}
	if cfg.EncryptionMasterKey == "" {
		cfg.EncryptionMasterKey = "0123456789abcdef0123456789abcdef"
	}
	if cfg.VectorProvider == "" {
		cfg.VectorProvider = "pgvector"
	}

	return &cfg, nil
}
