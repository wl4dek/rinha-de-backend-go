package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	fraudscore "rinha-de-backend/internal/frauds_core"
	ready "rinha-de-backend/internal/ready"
)

type Config struct {
	Port string
}

func loadConfig() Config {
	config := Config{
		Port: getEnv("PORT", "8080"),
	}
	return config
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/ready", ready.HandleReady)
	mux.HandleFunc("/fraud-score", fraudscore.HandleFraudScore)

	return mux
}

func main() {
	config := loadConfig()

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	mux := setupRoutes()

	log.Printf("Starting server on port %s", config.Port)

	err := http.ListenAndServe(":"+config.Port, mux)
	if err != nil {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}
}
