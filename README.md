# Gator

Gator ist ein CLI-basiertes Tool zum Aggregieren, Verwalten und Browsen von RSS-Feeds mit mehreren Benutzern. Dieses Projekt baut auf einen Kurs von `boot.dev` auf.

---

## REQUIREMENTS

- PostgreSQL
- [Goose](https://github.com/pressly/goose) für DB-Migrationen
- Go 1.20+
- sqlc

### Go Modules & Dependencies

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go get github.com/google/uuid
go get github.com/lib/pq
````

### Konfiguration

Lege eine `.gatorconfig.json` im Home-Verzeichnis an:

```json
{
  "db_url": "<Deine DB-URL>",
  "current_user_name": "<Dein UserName beim Login>"
}
```
---

## Installation

1. **Repository klonen**

```bash
git clone https://github.com/deinusername/gator.git
cd gator
```

2. **Binary bauen**

Diesen Befehl im root des Projekt-Verzeichnis ausführen.
```bash
go build -o gator
```

> Dadurch wird eine ausführbare Datei `gator` im aktuellen Ordner erzeugt.

---

## Nutzung

### Hilfe anzeigen

```bash
./gator help
```

### Durch Posts browsen

```bash
./gator browse
```

### Beiträge aggregieren

```bash
./gator agg
```


```bash
sudo mv gator /usr/local/bin/
gator help
```

---

## Datenbank

1. Migrationen ausführen:

```bash
goose postgres "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable" up
goose postgres "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable" down
```

2. SQLC-Code generieren:

```bash
sqlc generate
```

---

## Installation & Start

```bash
go run .
```

---

## Commands

Die CLI unterstützt folgende Commands:

### Allgemeine Commands

| Command                   | Beschreibung                                               |
| ------------------------- | ---------------------------------------------------------- |
| `help`                    | Zeigt alle verfügbaren Commands an                         |
| `exit`                    | Beendet die Anwendung                                      |
| `login <username>`        | Loggt einen existierenden User ein                         |
| `register <username>`     | Erstellt einen neuen User und setzt ihn als aktuellen User |
| `reset`                   | Löscht alle User aus der Datenbank                         |
| `users`                   | Listet alle User auf                                       |
| `feeds`                   | Listet alle Feeds auf                                      |
| `agg <time_between_reqs>` | Startet die Feed-Aggregation im angegebenen Intervall      |

### Commands für eingeloggte User

| Command                | Beschreibung                                                  |
| ---------------------- | ------------------------------------------------------------- |
| `addfeed <name> <url>` | Erstellt einen Feed und folgt diesem automatisch              |
| `follow <url>`         | Folgt einem existierenden Feed                                |
| `following`            | Listet alle Feeds auf, denen der User folgt                   |
| `unfollow <url>`       | Entfolgt einem Feed                                           |
| `browse [limit]`       | Zeigt die neuesten Posts von gefolgten Feeds an (Standard: 2) |

---

## Beispielworkflow

For jedem Befehl sollte `go run .` ergänzt werden.

```bash
# User registrieren und einloggen
register boots
login boots

# Feed hinzufügen
addfeed "TechCrunch" "https://techcrunch.com/feed/"

# Alle gefolgten Feeds anzeigen
following

# Aggregation starten (alle 10 Minuten)
agg 10m

# Neueste Posts anzeigen
browse 5

```

---

## Hinweise

* Alle Commands geben bei Fehlern eine entsprechende Fehlermeldung auf STDERR aus.
* `browse` zeigt "Keine Posts gefunden." an, wenn der User keine Posts in seinen abonnierten Feeds hat.
* `agg` kann mit Ctrl+C sauber beendet werden.
* Feed-Aggregation speichert doppelte Posts nicht erneut.

---

## Extending the Project

* You've done all the required steps, but if you'd like to make this project your own, here are some ideas:
* Add sorting and filtering options to the browse command
* Add pagination to the browse command
* Add concurrency to the agg command so that it can fetch more frequently
* Add a search command that allows for fuzzy searching of posts
* Add bookmarking or liking posts
* Add a TUI that allows you to select a post in the terminal and view it in a more readable format (either in the terminal or open in a browser)
* Add an HTTP API (and authentication/authorization) that allows other users to interact with the service remotely
* Write a service manager that keeps the agg command running in the background and restarts it if it crashes
