package db

import (
	"embed"
)

//go:embed migrations/*
var MigrationsFS embed.FS

// GetMigrationsFS returns the embedded FS containing the database migration files.
func GetMigrationsFS() embed.FS {
	return MigrationsFS
}
