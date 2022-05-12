package pg

import (
	"context"
	"database/sql"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/jackc/pgx/v4/stdlib" // driver
	"github.com/rendau/dop/adapters/db"
	"github.com/rendau/dop/adapters/logger"
)

const ErrPrefix = "pg-error"
const TransactionCtxKey = "pg_transaction"

type St struct {
	debug bool
	lg    logger.WarnAndError

	Con *pgxpool.Pool
}

type OptionsSt struct {
	Dsn               string
	Timezone          string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
	LazyConnect       bool
}

type conSt interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

type txContainerSt struct {
	tx pgx.Tx
}

var (
	queryParamRegexp = regexp.MustCompile(`(?si)\$\{[^}]+\}`)
)

func New(debug bool, lg logger.WarnAndError, opts OptionsSt) (*St, error) {
	cfg, err := opts.getConfig()
	if err != nil {
		lg.Errorw("Fail to create config", err, "opts", opts)
		return nil, err
	}

	dbPool, err := pgxpool.ConnectConfig(context.Background(), cfg)
	if err != nil {
		lg.Errorw(ErrPrefix+": Fail to connect to db", err)
		return nil, err
	}

	return &St{
		debug: debug,
		lg:    lg,
		Con:   dbPool,
	}, nil
}

func (o OptionsSt) getConfig() (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig(o.Dsn)
	if err != nil {
		return nil, err
	}

	// default values
	cfg.ConnConfig.RuntimeParams["timezone"] = "Asia/Almaty"
	cfg.MaxConns = 100
	cfg.MinConns = 5
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 15 * time.Minute
	cfg.HealthCheckPeriod = 20 * time.Second
	cfg.LazyConnect = true

	// customs
	if o.Timezone != "" {
		cfg.ConnConfig.RuntimeParams["timezone"] = o.Timezone
	}
	if o.MaxConns != 0 {
		cfg.MaxConns = o.MaxConns
	}
	if o.MinConns != 0 {
		cfg.MinConns = o.MinConns
	}
	if o.MaxConnLifetime != 0 {
		cfg.MaxConnLifetime = o.MaxConnLifetime
	}
	if o.MaxConnIdleTime != 0 {
		cfg.MaxConnIdleTime = o.MaxConnIdleTime
	}
	if o.HealthCheckPeriod != 0 {
		cfg.HealthCheckPeriod = o.HealthCheckPeriod
	}
	if o.LazyConnect {
		cfg.LazyConnect = o.LazyConnect
	}

	return cfg, nil
}

func (d *St) hErr(err error) error {
	switch err {
	case nil:
		return nil
	case pgx.ErrNoRows, sql.ErrNoRows:
		err = db.ErrNoRows
	default:
		d.lg.Errorw(ErrPrefix, err)
	}

	return err
}

func (d *St) getCon(ctx context.Context) conSt {
	if tx := d.getContextTransaction(ctx); tx != nil {
		return tx
	}
	return d.Con
}

// transaction

func (d *St) getContextTransactionContainer(ctx context.Context) *txContainerSt {
	contextV := ctx.Value(TransactionCtxKey)
	if contextV == nil {
		return nil
	}

	switch tx := contextV.(type) {
	case *txContainerSt:
		return tx
	default:
		return nil
	}
}

func (d *St) getContextTransaction(ctx context.Context) pgx.Tx {
	container := d.getContextTransactionContainer(ctx)
	if container != nil {
		return container.tx
	}

	return nil
}

func (d *St) ContextWithTransaction(ctx context.Context) (context.Context, error) {
	tx, err := d.Con.Begin(ctx)
	if err != nil {
		return ctx, d.hErr(err)
	}

	return context.WithValue(ctx, TransactionCtxKey, &txContainerSt{tx: tx}), nil
}

func (d *St) CommitContextTransaction(ctx context.Context) error {
	tx := d.getContextTransaction(ctx)
	if tx == nil {
		return nil
	}

	err := tx.Commit(ctx)
	if err != nil {
		if err != pgx.ErrTxClosed &&
			err != pgx.ErrTxCommitRollback {
			_ = tx.Rollback(ctx)

			return d.hErr(err)
		}
	}

	return nil
}

func (d *St) RollbackContextTransaction(ctx context.Context) {
	tx := d.getContextTransaction(ctx)
	if tx == nil {
		return
	}

	_ = tx.Rollback(ctx)
}

func (d *St) RenewContextTransaction(ctx context.Context) error {
	var err error

	container := d.getContextTransactionContainer(ctx)
	if container == nil {
		d.lg.Errorw(ErrPrefix+": Transaction container not found in context", nil)
		return nil
	}

	if container.tx != nil {
		err = container.tx.Commit(ctx)
		if err != nil {
			if err != pgx.ErrTxClosed &&
				err != pgx.ErrTxCommitRollback {
				_ = container.tx.Rollback(ctx)

				return d.hErr(err)
			}
		}
	}

	container.tx, err = d.Con.Begin(ctx)
	if err != nil {
		return d.hErr(err)
	}

	return nil
}

// query

func (d *St) DbExec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return d.getCon(ctx).Exec(ctx, sql, args...)
}

func (d *St) DbQuery(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return d.getCon(ctx).Query(ctx, sql, args...)
}

func (d *St) DbQueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return d.getCon(ctx).QueryRow(ctx, sql, args...)
}

func (d *St) queryRebindNamed(sql string, argMap map[string]interface{}) (string, []interface{}) {
	resultQuery := sql
	args := make([]interface{}, 0, len(argMap))

	for k, v := range argMap {
		if strings.Contains(resultQuery, "${"+k+"}") {
			args = append(args, v)
			resultQuery = strings.ReplaceAll(resultQuery, "${"+k+"}", "$"+strconv.Itoa(len(args)))
		}
	}

	if d.debug {
		if strings.Index(resultQuery, "${") > -1 {
			for _, x := range queryParamRegexp.FindAllString(resultQuery, 1) {
				d.lg.Errorw(ErrPrefix+": missing param", nil, "param", x, "query", resultQuery)
			}
		}
	}

	return resultQuery, args
}

func (d *St) DbExecM(ctx context.Context, sql string, argMap map[string]interface{}) (pgconn.CommandTag, error) {
	rbSql, args := d.queryRebindNamed(sql, argMap)

	return d.getCon(ctx).Exec(ctx, rbSql, args...)
}

func (d *St) DbQueryM(ctx context.Context, sql string, argMap map[string]interface{}) (pgx.Rows, error) {
	rbSql, args := d.queryRebindNamed(sql, argMap)

	return d.getCon(ctx).Query(ctx, rbSql, args...)
}

func (d *St) DbQueryRowM(ctx context.Context, sql string, argMap map[string]interface{}) pgx.Row {
	rbSql, args := d.queryRebindNamed(sql, argMap)

	return d.getCon(ctx).QueryRow(ctx, rbSql, args...)
}
