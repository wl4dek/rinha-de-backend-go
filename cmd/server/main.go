package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
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
	mux.Handle("/debug/pprof/", http.DefaultServeMux)

	return mux
}

func main() {
	config := loadConfig()

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	mux := setupRoutes()

	log.Printf("Starting server on port %s", config.Port)

	go func() {
		err := http.ListenAndServe(":"+config.Port, mux)
		if err != nil {
			log.Fatalf("Erro ao iniciar servidor: %v", err)
		}
	}()

	go func() {
		log.Println("pprof on :6060")
		log.Fatal(http.ListenAndServe("0.0.0.0:6060", nil))
	}()

	select {}
}
