package main

import (
	_ "github.com/joho/godotenv/autoload"
	"os"
	"strconv"
)

func envOrDefault(key string, fallback string) string {
	env := os.Getenv(key)
	if len(env) == 0 {
		return fallback
	}

	return env
}

func envIntOrDefault(key string, fallback int) int {
	env := os.Getenv(key)
	if len(env) == 0 {
		return fallback
	}

	val, err := strconv.Atoi(env)
	if err != nil {
		return fallback
	}

	return val
}
