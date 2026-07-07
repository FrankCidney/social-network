package config

import (
	"os"
	"strings"
)

func IsDevelopment() bool {
	env := firstEnv("APP_ENV", "GO_ENV", "ENV")
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "development", "dev", "local", "test":
		return true
	default:
		return false
	}
}

func SecureCookiesEnabled() bool {
	return !IsDevelopment()
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			return value
		}
	}
	return ""
}
