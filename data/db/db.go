package db

import (
	_ "github.com/lib/pq"
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
