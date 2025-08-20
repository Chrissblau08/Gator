package commands

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Chrissblau08/Gator/internal/database"
	"github.com/Chrissblau08/Gator/internal/rss"
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
			"agg":      HandlerAgg,
			"feeds":    HandlerFeeds,

			// Diese Handler brauchen den eingeloggten User → Middleware einhaken
			"addfeed":   MiddlewareLoggedIn(HandlerAddFeed),
			"follow":    MiddlewareLoggedIn(HandlerFollow),
			"following": MiddlewareLoggedIn(HandlerFollowing),
			"unfollow":  MiddlewareLoggedIn(HandlerUnFollow),
			"browse":    MiddlewareLoggedIn(HandlerBrowse),
		},
	}
}

// ==== MiddleWare ====

func MiddlewareLoggedIn(
	handler func(s *state.State, cmd Command, user database.User) error,
) func(s *state.State, cmd Command) error {
	return func(s *state.State, cmd Command) error {
		ctx := context.Background()

		user, err := s.DB.GetUser(ctx, s.Config.CurrentUserName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fehler: konnte User nicht finden (%v)\n", err)
			os.Exit(1)
		}

		// Aufruf des "inneren" Handlers mit User
		return handler(s, cmd, user)
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

func HandlerAgg(s *state.State, cmd Command) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("usage: agg <time_between_reqs>")
	}

	// Zeitintervall parsen
	timeBetweenRequests, err := time.ParseDuration(cmd.Args[0])
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	fmt.Printf("Collecting feeds every %s\n", timeBetweenRequests)

	// Ticker starten
	ticker := time.NewTicker(timeBetweenRequests)
	defer ticker.Stop()

	// Signal für sauberes Beenden (Ctrl+C)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	// Erste Ausführung sofort
	go func() {
		ScrapeFeeds(s)
	}()

	for {
		select {
		case <-ticker.C:
			ScrapeFeeds(s)
		case <-sigCh:
			fmt.Println("Stopping feed collection...")
			return nil
		case <-ctx.Done():
			return nil
		}
	}
}

func HandlerAddFeed(s *state.State, cmd Command, user database.User) error {
	if len(cmd.Args) < 2 {
		return fmt.Errorf("usage: addfeed <name of feed> <URL of feed>")
	}

	ctx := context.Background()
	now := time.Now()

	feedName := cmd.Args[0]
	feedURL := cmd.Args[1]

	feed, err := s.DB.CreateFeed(ctx,
		database.CreateFeedParams{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			Name:      feedName,
			Url:       feedURL,
			UserID:    user.ID,
		})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fehler: konnte Users nicht laden (%v)\n", err)
		os.Exit(1)
	}

	// Follow erstellen und Rückgabe nutzen
	feedFollow, err := s.DB.CreateFeedFollow(ctx, database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fehler: konnte Feed-Follow nicht erstellen (%v)\n", err)
		os.Exit(1)
	}

	fmt.Printf("Feed '%s' wurde erstellt und von '%s' automatisch gefolgt.\n", feed.Name, feedFollow.UserName)

	return nil
}

func HandlerFeeds(s *state.State, cmd Command) error {
	ctx := context.Background()

	feeds, err := s.DB.GetFeeds(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fehler: konnte keine Feeds finden (%v)\n", err)
		os.Exit(1)
	}

	for _, feed := range feeds {

		fmt.Println("Name: " + feed.Name)
		fmt.Println("URL: " + feed.Url)

		user, err := s.DB.GetUserByID(ctx, feed.UserID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fehler: konnte User nicht finden (%v)\n", err)
			os.Exit(1)
		}

		fmt.Println("Username: " + user.Name)
	}

	return nil
}

func HandlerFollow(s *state.State, cmd Command, user database.User) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("usage: follow <url>")
	}
	ctx := context.Background()
	now := time.Now()
	feedURL := cmd.Args[0]

	feed, err := s.DB.GetFeedByURL(ctx, feedURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fehler: konnte Feed nicht finden (%v)\n", err)
		os.Exit(1)
	}

	feedFollow, err := s.DB.CreateFeedFollow(ctx,
		database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			UserID:    user.ID,
			FeedID:    feed.ID,
		})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fehler: konnte Users nicht laden (%v)\n", err)
		os.Exit(1)
	}

	fmt.Printf("FeedFollow erstellt:\nFeed: %s\nUser: %s\n", feedFollow.FeedName, feedFollow.UserName)

	return nil
}

func HandlerFollowing(s *state.State, cmd Command, user database.User) error {

	ctx := context.Background()
	currentUserName := s.Config.CurrentUserName

	following, err := s.DB.GetFeedFollowsForUser(ctx, user.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fehler: konnte UserFeeds nicht finden (%v)\n", err)
		os.Exit(1)
	}

	fmt.Printf("Followed Feeds (current-user): %s\n", currentUserName)
	for _, follow_feed := range following {
		fmt.Println("Name: " + follow_feed.FeedName)
	}

	return nil
}

func HandlerUnFollow(s *state.State, cmd Command, user database.User) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("usage: unfollow <url>")
	}

	ctx := context.Background()
	url := cmd.Args[0]

	err := s.DB.DeleteFeedFollowByUserAndURL(ctx, database.DeleteFeedFollowByUserAndURLParams{
		ID:  user.ID,
		Url: url,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fehler beim Unfollow: %v\n", err)
		os.Exit(1)
	}

	return nil
}

// ==== Helper Functions ====
func ScrapeFeeds(s *state.State) error {
	ctx := context.Background()

	// 1. Nächsten Feed abrufen
	feed, err := s.DB.GetNextFeedToFetch(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fehler beim Abrufen des nächsten Feeds (%v)\n", err)
		os.Exit(1)
	}

	// 2. Feed als abgerufen markieren
	if err := s.DB.MarkFeedFetched(ctx, feed.ID); err != nil {
		fmt.Fprintf(os.Stderr, "Fehler beim Markieren des Feeds als abgerufen (%v)\n", err)
		os.Exit(1)
	}

	// 3. Feed-Daten abrufen
	rssFeed, err := rss.FetchFeed(ctx, feed.Url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fehler beim Abrufen des Feeds von URL %s (%v)\n", feed.Url, err)
		os.Exit(1)
	}

	// 4. Feed-Items iterieren und in DB speichern
	fmt.Printf("Feed: %s (%s)\n", feed.Name, feed.Url)
	for _, item := range rssFeed.Channel.Item {
		// published_at parsen
		var publishedAt time.Time
		if item.PubDate != "" {
			publishedAt, _ = time.Parse(time.RFC1123Z, item.PubDate) // Standardformat RFC1123Z
			if publishedAt.IsZero() {
				publishedAt, _ = time.Parse(time.RFC1123, item.PubDate)
			}
		}

		err := s.DB.CreatePost(ctx, database.CreatePostParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Title:     item.Title,
			Url:       item.Link,
			Description: sql.NullString{
				String: item.Description,
				Valid:  item.Description != "",
			},
			PublishedAt: sql.NullTime{Time: publishedAt, Valid: !publishedAt.IsZero()},
			FeedID:      feed.ID,
		})
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key") {
				continue
			}
			fmt.Fprintf(os.Stderr, "Fehler beim Speichern des Posts '%s' (%v)\n", item.Title, err)
		} else {
			fmt.Printf("Post gespeichert: %s\n", item.Title)
		}
	}

	return nil
}

func HandlerBrowse(s *state.State, cmd Command, user database.User) error {
	ctx := context.Background()

	// 1. Limit aus den Argumenten auslesen, Standard = 2
	limit := 2
	if len(cmd.Args) >= 1 {
		if l, err := strconv.Atoi(cmd.Args[0]); err == nil && l > 0 {
			limit = l
		} else {
			fmt.Fprintf(os.Stderr, "Ungültiger Limit-Wert '%s', benutze Standard %d\n", cmd.Args[0], limit)
		}
	}

	// 2. Posts aus der DB abrufen
	posts, err := s.DB.GetPostsForUser(ctx, database.GetPostsForUserParams{
		ID:    user.ID,
		Limit: int32(limit),
	})
	if err != nil {
		return fmt.Errorf("fehler beim Abrufen der Posts: %w", err)
	}

	// 3. Posts ausgeben
	if len(posts) == 0 {
		fmt.Println("Keine Posts gefunden.")
		return nil
	}

	for i, post := range posts {
		fmt.Printf("[%d] %s\n", i+1, post.Title)
		fmt.Printf("    URL: %s\n", post.Url)
		if post.Description.Valid {
			fmt.Printf("    Description: %s\n", post.Description.String)
		}
		if post.PublishedAt.Valid {
			fmt.Printf("    Published: %s\n", post.PublishedAt.Time.Format(time.RFC1123))
		}
		fmt.Println("---------------------------------------------------")
	}

	return nil
}
