package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/jackc/pgx/v4/stdlib" // driver
	"github.com/rendau/dop/adapters/db"
	"github.com/rendau/dop/adapters/logger"
)

type St struct {
	debug bool
	lg    logger.WarnAndError

	Con *pgxpool.Pool
}

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
		return ctx, d.HErr(err)
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

			return d.HErr(err)
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

				return d.HErr(err)
			}
		}
	}

	container.tx, err = d.Con.Begin(ctx)
	if err != nil {
		return d.HErr(err)
	}

	return nil
}

// query

func (d *St) DbExec(ctx context.Context, sql string, args ...interface{}) error {
	_, err := d.getCon(ctx).Exec(ctx, sql, args...)
	return d.HErr(err)
}

func (d *St) DbQuery(ctx context.Context, sql string, args ...interface{}) (Rows, error) {
	rows, err := d.getCon(ctx).Query(ctx, sql, args...)
	return rowsSt{Rows: rows, db: d}, d.HErr(err)
}

func (d *St) DbQueryRow(ctx context.Context, sql string, args ...interface{}) Row {
	return rowSt{Row: d.getCon(ctx).QueryRow(ctx, sql, args...), db: d}
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

func (d *St) DbExecM(ctx context.Context, sql string, argMap map[string]interface{}) error {
	rbSql, args := d.queryRebindNamed(sql, argMap)
	_, err := d.getCon(ctx).Exec(ctx, rbSql, args...)
	return d.HErr(err)
}

func (d *St) DbQueryM(ctx context.Context, sql string, argMap map[string]interface{}) (Rows, error) {
	rbSql, args := d.queryRebindNamed(sql, argMap)
	rows, err := d.getCon(ctx).Query(ctx, rbSql, args...)
	return rowsSt{Rows: rows, db: d}, d.HErr(err)
}

func (d *St) DbQueryRowM(ctx context.Context, sql string, argMap map[string]interface{}) Row {
	rbSql, args := d.queryRebindNamed(sql, argMap)
	return rowSt{Row: d.getCon(ctx).QueryRow(ctx, rbSql, args...), db: d}
}

func (d *St) ValidateColNames(names []string, allowed map[string]bool) ([]string, error) {
	for _, col := range names {
		if !allowed[col] {
			return nil, d.HErr(fmt.Errorf("%w: '%s'", db.ErrBadColumnName, col))
		}
	}

	if len(names) == 0 {
		for k := range allowed {
			names = append(names, k)
		}
	}

	return names, nil
}

func (d *St) HErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, pgx.ErrNoRows), errors.Is(err, sql.ErrNoRows):
		err = db.ErrNoRows
	default:
		d.lg.Errorw(ErrPrefix, err)
	}

	return err
}
