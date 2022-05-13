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
	DbExec(ctx context.Context, sql string, args ...interface{}) error
	DbQuery(ctx context.Context, sql string, args ...interface{}) (Rows, error)
	DbQueryRow(ctx context.Context, sql string, args ...interface{}) Row
	DbExecM(ctx context.Context, sql string, argMap map[string]interface{}) error
	DbQueryM(ctx context.Context, sql string, argMap map[string]interface{}) (Rows, error)
	DbQueryRowM(ctx context.Context, sql string, argMap map[string]interface{}) Row
	HErr(err error) error
}

type ConnectionWithHelpers interface {
	Connection

	HfList(dst any, tables, conds []string, lPars types.ListParams, allowedCols map[string]string) error
	HfGenerateColumns(rNames []string, allowed map[string]string) ([]string, []string)
	HfGetCUFields(obj interface{}) map[string]interface{}
	HfCreate(ctx context.Context, table string, obj interface{}, retCol string, retV interface{}) error
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
	Scan(dest ...interface{}) error
}

type Row interface {
	Scan(dest ...interface{}) error
}

type conSt interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}
