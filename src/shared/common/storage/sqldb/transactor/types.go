package transactor

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// DBTX is the common interface between *[sqlx.DB] and *[sqlx.Tx].
type DBTX interface {
	// database/sql methods

	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row

	Exec(query string, args ...any) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row

	// sqlx methods

	GetContext(ctx context.Context, dest any, query string, args ...any) error
	MustExecContext(ctx context.Context, query string, args ...any) sql.Result
	NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error)
	PrepareNamedContext(ctx context.Context, query string) (*sqlx.NamedStmt, error)
	PreparexContext(ctx context.Context, query string) (*sqlx.Stmt, error)
	QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row
	QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error)
	SelectContext(ctx context.Context, dest any, query string, args ...any) error

	Get(dest any, query string, args ...any) error
	MustExec(query string, args ...any) sql.Result
	NamedExec(query string, arg any) (sql.Result, error)
	NamedQuery(query string, arg any) (*sqlx.Rows, error)
	PrepareNamed(query string) (*sqlx.NamedStmt, error)
	Preparex(query string) (*sqlx.Stmt, error)
	QueryRowx(query string, args ...any) *sqlx.Row
	Queryx(query string, args ...any) (*sqlx.Rows, error)
	Select(dest any, query string, args ...any) error

	Rebind(query string) string
	BindNamed(query string, arg any) (string, []any, error)
	DriverName() string
}

type sqlxDB interface {
	DBTX
	BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error)
}

type sqlxTx interface {
	Commit() error
	Rollback() error
}

var (
	_ DBTX   = &sqlx.DB{}
	_ DBTX   = &sqlx.Tx{}
	_ sqlxDB = &sqlx.DB{}
	_ sqlxTx = &sqlx.Tx{}
)

type (
	transactorKey struct{}
	// DBContext is used to get the current DB handler from the context.
	// It returns the current transaction if there is one, otherwise it will return the original DB.
	DBContext func(context.Context) DBTX
)

func txToContext(ctx context.Context, tx sqlxDB) context.Context {
	return context.WithValue(ctx, transactorKey{}, tx)
}

func txFromContext(ctx context.Context) sqlxDB {
	if tx, ok := ctx.Value(transactorKey{}).(sqlxDB); ok {
		return tx
	}
	return nil
}
