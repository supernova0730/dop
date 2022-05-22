package pg

import (
	"context"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/rendau/dop/types"
)

type Full interface {
	Connection
	ContextTransaction
}

type Connection interface {
	DbExec(ctx context.Context, sql string, args ...any) error
	DbQuery(ctx context.Context, sql string, args ...any) (Rows, error)
	DbQueryRow(ctx context.Context, sql string, args ...any) Row
	DbExecM(ctx context.Context, sql string, argMap map[string]any) error
	DbQueryM(ctx context.Context, sql string, argMap map[string]any) (Rows, error)
	DbQueryRowM(ctx context.Context, sql string, argMap map[string]any) Row
	HErr(err error) error
}

type ConnectionWithHelpers interface {
	Connection

	HfList(
		ctx context.Context,
		dst any,
		tables, conds []string,
		args map[string]any,
		lPars types.ListParams,
		allowedCols map[string]string,
		allowedSorts map[string]string,
		allowedSortNames map[string]string,
	) (int64, error)
	HfGenerateColumns(rNames []string, allowed map[string]string) ([]string, []string)
	HfGenerateSort(rNames []string, allowed map[string]string) []string
	HfGet(ctx context.Context, dst any, tables, conds []string, args map[string]any, allowedCols map[string]string) error
	HfCreate(ctx context.Context, table string, obj any, retCol string, retV any) error
	HfUpdate(ctx context.Context, table string, obj any, conds []string, condArgs map[string]any) error
	HfGetCUFields(obj any) map[string]any
	HfDelete(ctx context.Context, table string, conds []string, args map[string]any) error
}

type ContextTransaction interface {
	ContextWithTransaction(ctx context.Context) (context.Context, error)
	CommitContextTransaction(ctx context.Context) error
	RollbackContextTransaction(ctx context.Context)
	RenewContextTransaction(ctx context.Context) error
}

type Rows interface {
	Close()
	Err() error
	Next() bool
	Scan(dest ...any) error
}

type Row interface {
	Scan(dest ...any) error
}

type conSt interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
