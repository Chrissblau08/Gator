package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config repräsentiert die JSON-Struktur deiner Konfigurationsdatei
type Config struct {
	DB_URL          string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
	// Hier kannst du später weitere Felder hinzufügen, z.B. Tokens, Ports etc.
}

// getConfigPath liefert den Pfad zur Config-Datei (~/.gatorconfig.json)
func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("konnte HOME-Verzeichnis nicht finden: %w", err)
	}
	return filepath.Join(home, ".gatorconfig.json"), nil
}

// Read liest die Config aus ~/.gatorconfig.json
func Read() (*Config, error) {
	path, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("konnte Config-Datei nicht öffnen: %w", err)
	}
	defer file.Close()

	var cfg Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("konnte Config nicht dekodieren: %w", err)
	}

	return &cfg, nil
}

// write speichert die Config nach ~/.gatorconfig.json
func (c *Config) write() error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("konnte Config-Datei nicht schreiben: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(c)
}

// SetUser setzt den aktuellen User und speichert die Config
func (c *Config) SetUser(username string) error {
	c.CurrentUserName = username
	return c.write()
}

// Print gibt alle Felder der Config auf der Konsole aus
func (c *Config) Print() {
	fmt.Println("===== Aktuelle Config =====")
	fmt.Printf("DB_URL: %s\n", c.DB_URL)
	fmt.Printf("CurrentUserName: %s\n", c.CurrentUserName)
	fmt.Println("===========================")
}
