package pg

import (
	"context"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

type Full interface {
	Connection
	ContextTransaction
}

type Connection interface {
	DbExec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
	DbQuery(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	DbQueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	DbExecM(ctx context.Context, sql string, argMap map[string]interface{}) (pgconn.CommandTag, error)
	DbQueryM(ctx context.Context, sql string, argMap map[string]interface{}) (pgx.Rows, error)
	DbQueryRowM(ctx context.Context, sql string, argMap map[string]interface{}) pgx.Row
}

type ContextTransaction interface {
	ContextWithTransaction(ctx context.Context) (context.Context, error)
	CommitContextTransaction(ctx context.Context) error
	RollbackContextTransaction(ctx context.Context)
	RenewContextTransaction(ctx context.Context) error
}
