package app

import (
	"flag"
	"net/url"
	"os"
)

type Config struct {
	RunAddress           string
	DatabaseURI          string
	AccrualSystemAddress string
	LogLevel             string
	JWTSecretKey         string
	MigrationsPath       string
}

func NewConfigFromFlags() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.RunAddress, "a", "localhost:8080", "Server address (env: RUN_ADDRESS)")
	flag.StringVar(&cfg.DatabaseURI, "d", "", "Database URI (env: DATABASE_URI)")
	flag.StringVar(&cfg.AccrualSystemAddress, "r", "", "Accrual system address (env: ACCRUAL_SYSTEM_ADDRESS)")
	flag.StringVar(&cfg.LogLevel, "l", "debug", "Log level (debug|info|warn|error) (env: LOG_LEVEL)")
	flag.StringVar(&cfg.JWTSecretKey, "jwt-secret", "", "JWT secret key (env: JWT_SECRET_KEY)")
	flag.StringVar(&cfg.MigrationsPath, "migrations", "./migrations", "Path to migrations folder (env:MIGRATIONS_PATH)")
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
	u, err := url.Parse(c.DatabaseURI)
	if err != nil {
		return c.DatabaseURI
	}

	if u.User != nil {
		if _, hasPassword := u.User.Password(); hasPassword {
			u.User = url.UserPassword(u.User.Username(), "***")
		}
	}
	return u.String()
}
