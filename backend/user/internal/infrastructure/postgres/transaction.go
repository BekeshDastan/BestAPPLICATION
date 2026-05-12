package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type txKey struct{}

// DB wraps sqlx.DB and implements domain.Transactor.
type DB struct {
	db *sqlx.DB
}

func NewDB(db *sqlx.DB) *DB { return &DB{db: db} }

func (d *DB) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := d.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback after %w: %v", err, rbErr)
		}
		return err
	}
	return tx.Commit()
}

// querier returns the active tx from context, or the plain db.
func querier(ctx context.Context, db *sqlx.DB) sqlx.ExtContext {
	if tx, ok := ctx.Value(txKey{}).(*sqlx.Tx); ok {
		return tx
	}
	return db
}
