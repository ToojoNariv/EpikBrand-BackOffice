package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Config stocke tous les paramètres de configuration de l'application
type Config struct {
	DBHost            string
	DBPort            string
	DBUser            string
	DBPassword        string
	DBName            string
	DBSSLMode         string
	ServerPort        string
	RemoteURL         string
	GoogleClientID    string
	InitialAdminEmail string
}

// LoadConfig charge la configuration à partir des variables d'environnement ou d'un fichier .env
func LoadConfig() (*Config, error) {
	// Tente de charger le fichier .env s'il existe
	_ = LoadEnvFile(".env")

	cfg := &Config{
		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnv("DB_PORT", "5432"),
		DBUser:            getEnv("DB_USER", "postgres"),
		DBPassword:        getEnv("DB_PASSWORD", "postgres"),
		DBName:            getEnv("DB_NAME", "epikbrandadmin"),
		DBSSLMode:         getEnv("DB_SSLMODE", "disable"),
		ServerPort:        getEnv("PORT", "8080"),
		RemoteURL:         getEnv("REMOTE_URL", ""),
		GoogleClientID:    getEnv("GOOGLE_CLIENT_ID", ""),
		InitialAdminEmail: getEnv("INITIAL_ADMIN_EMAIL", ""),
	}

	return cfg, nil
}

// getEnv récupère une variable d'environnement ou retourne une valeur par défaut si elle n'est pas définie
func getEnv(key, defaultValue string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultValue
}

// LoadEnvFile charge les paires clé=valeur d'un fichier et les définit comme variables d'environnement
func LoadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err // Le fichier n'existe pas ou ne peut pas être ouvert, on l'ignore silencieusement
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		
		// Enlever d'éventuels guillemets entourant la valeur
		val = strings.Trim(val, `"'`)
		
		_ = os.Setenv(key, val)
	}
	return scanner.Err()
}

// GetDSN génère la chaîne de connexion (Data Source Name) pour PostgreSQL en échappant les valeurs spéciales
func (c *Config) GetDSN() string {
	escape := func(s string) string {
		s = strings.ReplaceAll(s, "\\", "\\\\")
		s = strings.ReplaceAll(s, "'", "\\'")
		return s
	}
	return fmt.Sprintf("host='%s' port='%s' user='%s' password='%s' dbname='%s' sslmode='%s'",
		escape(c.DBHost), escape(c.DBPort), escape(c.DBUser), escape(c.DBPassword), escape(c.DBName), escape(c.DBSSLMode))
}
