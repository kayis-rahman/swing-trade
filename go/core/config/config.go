package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// Config contains the runtime settings for the no-traffic platform skeleton.
type Config struct {
	HTTPAddr                 string
	APIKey                   string
	APIKeyEnabled            bool
	PublicRateLimitPerMinute int
	CORSAllowedOrigins       []string
	DatabaseURL              string
}

// Load reads the established Java deployment environment names through lookup.
func Load(lookup func(string) (string, bool)) (Config, error) {
	port := "8081"
	if value, ok := lookup("SERVER_PORT"); ok && strings.TrimSpace(value) != "" {
		port = strings.TrimSpace(value)
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return Config{}, fmt.Errorf("SERVER_PORT must be an integer from 1 to 65535")
	}

	apiKeyEnabled := false
	if value, ok := lookup("API_KEY_AUTH_ENABLED"); ok && strings.TrimSpace(value) != "" {
		apiKeyEnabled, err = strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return Config{}, fmt.Errorf("API_KEY_AUTH_ENABLED must be a boolean: %w", err)
		}
	}
	apiKey, _ := lookup("API_KEY")
	if apiKeyEnabled && apiKey == "" {
		return Config{}, fmt.Errorf("API_KEY is required when API_KEY_AUTH_ENABLED is true")
	}

	limit := 100
	if value, ok := lookup("API_PUBLIC_RATE_LIMIT_PER_MINUTE"); ok && strings.TrimSpace(value) != "" {
		limit, err = strconv.Atoi(strings.TrimSpace(value))
		if err != nil || limit < 1 {
			return Config{}, fmt.Errorf("API_PUBLIC_RATE_LIMIT_PER_MINUTE must be a positive integer")
		}
	}

	origins := []string{"http://localhost:3000", "http://localhost:8080"}
	if value, ok := lookup("CORS_ALLOWED_ORIGINS"); ok {
		origins = splitOrigins(value)
	}
	databaseURL, _ := lookup("DATABASE_URL")

	host := "127.0.0.1"
	if value, ok := lookup("HTTP_HOST"); ok && strings.TrimSpace(value) != "" {
		host = strings.TrimSpace(value)
	}
	return Config{
		HTTPAddr: net.JoinHostPort(host, strconv.Itoa(portNumber)),

		APIKey:                   apiKey,
		APIKeyEnabled:            apiKeyEnabled,
		PublicRateLimitPerMinute: limit,
		CORSAllowedOrigins:       origins,
		DatabaseURL:              strings.TrimSpace(databaseURL),
	}, nil
}

func splitOrigins(value string) []string {
	seen := make(map[string]struct{})
	origins := make([]string, 0)
	for _, origin := range strings.Split(value, ",") {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		if _, ok := seen[origin]; ok {
			continue
		}
		seen[origin] = struct{}{}
		origins = append(origins, origin)
	}
	return origins
}
