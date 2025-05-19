package config

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Config struct {
	Env      string
	Port     string
	Debug    bool
	Database *Database
	Google   *GoogleOAuth
	Admin    *AdminConfig
	MAIlAPI  string
}

type Database struct {
	Host     string
	User     string
	Database string
	Password string
	Port     string
}

type GoogleOAuth struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type AdminConfig struct {
	ADMIN          string
	ADMIN_PASSWORD string
}

func (g *GoogleOAuth) Oauth() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     g.ClientID,
		ClientSecret: g.ClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  g.RedirectURL,
		Scopes:       []string{"email", "profile"},
	}
}

var (
	cfg  *Config
	once sync.Once
)

func Load() *Config {
	once.Do(func() {
		_ = godotenv.Load()

		cfg = &Config{
			Env:     getEnv("APP_ENV", "development"),
			Port:    getEnv("PORT", "8080"),
			Debug:   getEnvAsBool("DEBUG", true),
			MAIlAPI: getEnv("RESEND_API", ""),
			Database: &Database{
				Host:     getEnv("DB_HOST", "localhost"),
				User:     getEnv("DB_USER", "user"),
				Database: getEnv("DB_NAME", "dbname"),
				Password: getEnv("DB_PASSWORD", "dbpassword"),
				Port:     getEnv("DB_PORT", "5432"),
			},
			Google: &GoogleOAuth{
				ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
				ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:80/auth/google/callback"),
			},
			Admin: &AdminConfig{
				ADMIN:          getEnv("Admin", "djfdsjkjk"),
				ADMIN_PASSWORD: getEnv("djjdj", "djqkdj"),
			},
		}
	})
	return cfg
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvAsBool(name string, defaultVal bool) bool {
	valStr := os.Getenv(name)
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}
	return defaultVal
}

func (d *Database) String() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", d.User, d.Password, d.Host, d.Port, d.Database)
}
