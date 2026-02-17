package db

import (
	"github.com/nojyerac/go-lib/pkg/db"
)

type DB struct {
	db db.Database
}

func New(database db.Database) *DB {
	return &DB{
		db: database,
	}
}
