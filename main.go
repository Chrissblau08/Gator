package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/Chrissblau08/Gator/internal/commands"
	"github.com/Chrissblau08/Gator/internal/config"
	"github.com/Chrissblau08/Gator/internal/database"
	"github.com/Chrissblau08/Gator/internal/state"
	_ "github.com/lib/pq"
)

func main() {
	// 1. Config einlesen
	cfg, err := config.Read()
	if err != nil {
		log.Printf("Keine vorhandene Config oder Fehler beim Lesen: %v", err)
		cfg = &config.Config{} // leere Config anlegen
	}

	db, err := sql.Open("postgres", cfg.DB_URL)
	if err != nil {
		fmt.Println(err)
	}
	dbQueries := database.New(db)

	cfg.Print()

	// 2. State initialisieren
	s := &state.State{
		DB:     dbQueries,
		Config: cfg,
	}

	// 3. CLI-Argumente auslesen
	if len(os.Args) < 2 {
		fmt.Println("Usage: <command> [args...]")
		os.Exit(1) // Exit Code 1 bei zu wenig Argumenten
	}

	cmdName := os.Args[1]
	args := os.Args[2:]

	cmd := commands.Command{
		Name: cmdName,
		Args: args,
	}

	// 4. Command über zentrale Registry ausführen
	if err := commands.Registry.Run(s, cmd); err != nil {
		fmt.Printf("Fehler: %v\n", err)
		os.Exit(1) // Exit Code 1 bei Fehler
	}
}
