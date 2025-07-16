package app

import (
	"flag"
	"os"
	"strings"
)

type Config struct {
	RunAddress           string
	DatabaseURI          string
	AccrualSystemAddress string
	LogLevel             string
}

func NewConfigFromFlags() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.RunAddress, "a", "localhost:8080", "Server address (env: RUN_ADDRESS)")
	flag.StringVar(&cfg.DatabaseURI, "d", "", "Database URI (env: DATABASE_URI)")
	flag.StringVar(&cfg.AccrualSystemAddress, "r", "", "Accrual system address (env: ACCRUAL_SYSTEM_ADDRESS)")
	flag.StringVar(&cfg.LogLevel, "l", "debug", "Log level (debug|info|warn|error) (env: LOG_LEVEL)")
	flag.Parse()

	cfg.applyEnvVars()
	cfg.validate()

	return cfg
}

func (c *Config) applyEnvVars() {
	if envAddr := os.Getenv("RUN_ADDRESS"); envAddr != "" {
		c.RunAddress = envAddr
	}
	if envDB := os.Getenv("DATABASE_URI"); envDB != "" {
		c.DatabaseURI = envDB
	}
	if envAccrual := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); envAccrual != "" {
		c.AccrualSystemAddress = envAccrual
	}
	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		c.LogLevel = envLogLevel
	}
}

func (c *Config) validate() {
	if c.DatabaseURI == "" {
		panic("Database URI is required (use -d flag or DATABASE_URI env)")
	}
}

func (c *Config) MaskDBPassword() string {
	dsn := c.DatabaseURI
	if strings.Contains(dsn, "@") {
		parts := strings.Split(dsn, "@")
		if len(parts) > 1 {
			auth := parts[0]
			if strings.Contains(auth, ":") {
				userPass := strings.Split(auth, ":")
				if len(userPass) > 1 {
					return strings.Join([]string{userPass[0], "***", parts[1]}, "@")
				}
			}
		}
	}
	return dsn
}
