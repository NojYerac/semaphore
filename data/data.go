package data

import "context"

type Source interface {
	GetFlags(ctx context.Context) ([]Flag, error)
}

type Flag struct {
	ID      int    `db:"id" json:"id"`
	Name    string `db:"name" json:"name"`
	Enabled bool   `db:"enabled" json:"enabled"`
}
