package commands

import (
	"fmt"
	"os"
	"sort"

	"github.com/Chrissblau08/Gator/internal/state"
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
			"help":  HandlerHelp,
			"exit":  HandlerExit,
			"login": HandlerLogin,
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
	if err := s.Config.SetUser(username); err != nil {
		return err
	}

	fmt.Printf("User erfolgreich gesetzt auf: %s\n", username)
	return nil
}
