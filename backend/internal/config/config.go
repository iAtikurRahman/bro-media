package config

import "os"

type Config struct {
	Port           string
	TURNHost       string
	TURNPort       string
	TURNUser       string
	TURNPassword   string
	STUNHost       string
	AllowedOrigins []string
}

func Load() *Config {
	return &Config{
		Port:         getEnv("PORT", "8080"),
		TURNHost:     getEnv("TURN_HOST", "turn-server"),
		TURNPort:     getEnv("TURN_PORT", "3478"),
		TURNUser:     getEnv("TURN_USER", "bromedia"),
		TURNPassword: getEnv("TURN_PASSWORD", "bromedia-secret"),
		STUNHost:     getEnv("STUN_HOST", "stun:stun.l.google.com:19302"),
		AllowedOrigins: []string{
			getEnv("ALLOWED_ORIGIN", "*"),
		},
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
