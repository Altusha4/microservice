package email

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
	"strings"
)

// #######################################
// FACTORY
// #######################################
//
// Build wires up the concrete EmailSender based on PROVIDER_MODE.
// All providers live behind the same interface — switching is a
// one-env-var change, no recompilation.
//
// Modes:
//   SIMULATED  — deterministic-ish in-process fake, default
//   SMTP       — real SMTP via SMTP_* env vars (Mailtrap, etc.)

func Build() (EmailSender, error) {
	mode := strings.ToUpper(getEnv("PROVIDER_MODE", "SIMULATED"))
	switch mode {
	case "SIMULATED":
		latencyMs := getEnvInt("SIMULATOR_LATENCY_MS", 200)
		failure, _ := strconv.ParseFloat(getEnv("SIMULATOR_FAILURE_RATE", "0.3"), 64)
		log.Printf("[email] using SIMULATED provider (latency=%dms, failure_rate=%.2f)",
			latencyMs, failure)
		return NewSimulatedSender(
			time.Duration(latencyMs)*time.Millisecond,
			failure,
		), nil

	case "SMTP":
		cfg := SMTPConfig{
			Host:     getEnv("SMTP_HOST", ""),
			Port:     getEnv("SMTP_PORT", "2525"),
			Username: getEnv("SMTP_USERNAME", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", "no-reply@microservice.local"),
		}
		if cfg.Host == "" || cfg.Username == "" || cfg.Password == "" {
			return nil, fmt.Errorf("SMTP mode requires SMTP_HOST, SMTP_USERNAME, SMTP_PASSWORD")
		}
		log.Printf("[email] using SMTP provider (host=%s:%s)", cfg.Host, cfg.Port)
		return NewSMTPSender(cfg), nil

	default:
		return nil, fmt.Errorf("unknown PROVIDER_MODE=%q (want SIMULATED or SMTP)", mode)
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
