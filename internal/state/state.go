package state

import (
	"github.com/Chrissblau08/Gator/internal/config"
	"github.com/Chrissblau08/Gator/internal/database"
)

type State struct {
	DB     *database.Queries
	Config *config.Config
}
