package commands

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/Chrissblau08/Gator/internal/database"
	"github.com/Chrissblau08/Gator/internal/state"
	"github.com/google/uuid"
)

// Command enthält Name + Argumente
type Command struct {
	Name string
	Args []string
}

// CommandFunc ist die Signatur aller Handler
type CommandFunc func(s *state.State, cmd Command) error

// Commands hält alle bekannten Befehle
type Commands struct {
	Handlers map[string]CommandFunc
}

var Registry Commands

func init() {
	Registry = Commands{
		Handlers: map[string]CommandFunc{
			"help":     HandlerHelp,
			"exit":     HandlerExit,
			"login":    HandlerLogin,
			"register": HandlerRegister,
			"reset":    HandlerReset,
			"users":    HandlerUsers,
		},
	}
}

// ==== Methoden auf Commands ====

func (c *Commands) Run(s *state.State, cmd Command) error {
	handler, ok := c.Handlers[cmd.Name]
	if !ok {
		return fmt.Errorf("unbekannter Befehl: %s", cmd.Name)
	}
	return handler(s, cmd)
}

func (c *Commands) Register(name string, f CommandFunc) {
	c.Handlers[name] = f
}

// ==== Handler ====

func HandlerHelp(s *state.State, cmd Command) error {
	fmt.Println("Available commands:")

	// Namen sortieren für saubere Ausgabe
	names := make([]string, 0, len(Registry.Handlers))
	for name := range Registry.Handlers {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		fmt.Printf(" - %s\n", name)
	}
	return nil
}

func HandlerExit(s *state.State, cmd Command) error {
	fmt.Println("Goodbye!")
	os.Exit(0)
	return nil
}

func HandlerLogin(s *state.State, cmd Command) error {
	if len(cmd.Args) == 0 {
		return fmt.Errorf("usage: login <username>")
	}

	username := cmd.Args[0]
	ctx := context.Background()

	_, err := s.DB.GetUser(ctx, username)
	if err != nil {
		// Wenn der Name schon existiert → Exit mit 1
		fmt.Fprintf(os.Stderr, "Fehler: konnte User nicht finden (%v)\n", err)
		os.Exit(1)
	}

	if err := s.Config.SetUser(username); err != nil {
		return err
	}

	fmt.Printf("User erfolgreich gesetzt auf: %s\n", username)
	return nil
}

func HandlerRegister(s *state.State, cmd Command) error {
	// 1. Prüfen, ob ein Name mitgegeben wurde
	if len(cmd.Args) < 1 {
		return fmt.Errorf("usage: register <name>")
	}
	name := cmd.Args[0]

	// 2. User anlegen
	ctx := context.Background()
	now := time.Now()

	user, err := s.DB.CreateUser(ctx,
		database.CreateUserParams{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			Name:      name,
		},
	)
	if err != nil {
		// Wenn der Name schon existiert → Exit mit 1
		fmt.Fprintf(os.Stderr, "Fehler: konnte User nicht anlegen (%v)\n", err)
		os.Exit(1)
	}

	// 3. Aktuellen User in Config speichern
	if err := s.Config.SetUser(name); err != nil {
		return fmt.Errorf("konnte User nicht in Config speichern: %w", err)
	}

	// 4. Ausgabe für den Nutzer
	fmt.Printf("User %q wurde erfolgreich erstellt und als aktueller User gesetzt.\n", name)

	// 5. Debug-Ausgabe
	log.Printf("DEBUG: User %+v\n", user)

	return nil
}

func HandlerReset(s *state.State, cmd Command) error {
	ctx := context.Background()
	err := s.DB.ResetUsers(ctx)
	if err != nil {
		// Wenn der Name schon existiert → Exit mit 1
		fmt.Fprintf(os.Stderr, "Fehler: konnte Users nicht löschen (%v)\n", err)
		os.Exit(1)
	}

	return nil
}

func HandlerUsers(s *state.State, cmd Command) error {
	ctx := context.Background()

	users, err := s.DB.GetUsers(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fehler: konnte Users nicht laden (%v)\n", err)
		os.Exit(1)
	}

	for _, user := range users {
		line := "* " + user.Name
		if user.Name == s.Config.CurrentUserName {
			line += " (current)"
		}
		fmt.Println(line)
	}

	return nil
}
