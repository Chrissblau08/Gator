# Gator

## REQUIREMENTS
> postgresql
> goose
> go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

> go get github.com/google/uuid
> go get github.com/lib/pq
> .gatorconfig.json im Home mit den Attributen User und DB URL
```
{
  "db_url": "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable",
  "current_user_name": "boots"
}
`` 
## Mirgration

> goose postgres postgres://postgres:postgres@localhost:5432/gator up/down
> sqlc generate

## Commands