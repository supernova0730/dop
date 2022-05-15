package pg

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/jackc/pgx/v4/stdlib" // driver
	"github.com/rendau/dop/adapters/logger"
	"github.com/rendau/dop/errs"
	"github.com/rendau/dop/types"
)

type St struct {
	debug bool
	lg    logger.WarnAndError

	opts OptionsSt
	Con  *pgxpool.Pool
}

func New(debug bool, lg logger.WarnAndError, opts OptionsSt) (*St, error) {
	opts.mergeWithDefaults()

	cfg, err := pgxpool.ParseConfig(opts.Dsn)
	if err != nil {
		lg.Errorw("Fail to create config", err, "opts", opts)
		return nil, err
	}

	cfg.ConnConfig.RuntimeParams["timezone"] = opts.Timezone
	cfg.MaxConns = opts.MaxConns
	cfg.MinConns = opts.MinConns
	cfg.MaxConnLifetime = opts.MaxConnLifetime
	cfg.MaxConnIdleTime = opts.MaxConnIdleTime
	cfg.HealthCheckPeriod = opts.HealthCheckPeriod
	cfg.LazyConnect = true

	dbPool, err := pgxpool.ConnectConfig(context.Background(), cfg)
	if err != nil {
		lg.Errorw(ErrPrefix+": Fail to connect to db", err)
		return nil, err
	}

	return &St{
		debug: debug,
		lg:    lg,
		opts:  opts,
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

func (d *St) DbExec(ctx context.Context, sql string, args ...any) error {
	_, err := d.getCon(ctx).Exec(ctx, sql, args...)
	return d.HErr(err)
}

func (d *St) DbQuery(ctx context.Context, sql string, args ...any) (Rows, error) {
	rows, err := d.getCon(ctx).Query(ctx, sql, args...)
	return rowsSt{Rows: rows, db: d}, d.HErr(err)
}

func (d *St) DbQueryRow(ctx context.Context, sql string, args ...any) Row {
	return rowSt{Row: d.getCon(ctx).QueryRow(ctx, sql, args...), db: d}
}

func (d *St) queryRebindNamed(sql string, argMap map[string]any) (string, []any) {
	resultQuery := sql
	args := make([]any, 0, len(argMap))

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

func (d *St) DbExecM(ctx context.Context, sql string, argMap map[string]any) error {
	rbSql, args := d.queryRebindNamed(sql, argMap)
	_, err := d.getCon(ctx).Exec(ctx, rbSql, args...)
	return d.HErr(err)
}

func (d *St) DbQueryM(ctx context.Context, sql string, argMap map[string]any) (Rows, error) {
	rbSql, args := d.queryRebindNamed(sql, argMap)
	rows, err := d.getCon(ctx).Query(ctx, rbSql, args...)
	return rowsSt{Rows: rows, db: d}, d.HErr(err)
}

func (d *St) DbQueryRowM(ctx context.Context, sql string, argMap map[string]any) Row {
	rbSql, args := d.queryRebindNamed(sql, argMap)
	return rowSt{Row: d.getCon(ctx).QueryRow(ctx, rbSql, args...), db: d}
}

func (d *St) HErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, pgx.ErrNoRows), errors.Is(err, sql.ErrNoRows):
		err = errs.NoRows
	default:
		d.lg.Errorw(ErrPrefix, err)
	}

	return err
}

// helpers

func (d *St) HfList(
	ctx context.Context,
	dst any,
	tables, conds []string,
	args map[string]any,
	lPars types.ListParams,
	allowedCols map[string]string,
	allowedSorts map[string]string,
	allowedSortNames map[string]string,
) (int64, error) {
	var tCount int64

	if (lPars.WithTotalCount && lPars.PageSize > 0) || lPars.OnlyCount {
		err := d.DbQueryRowM(ctx, `select count(*)`+
			` from `+strings.Join(tables, " ")+
			` where `+strings.Join(conds, " and "), args).Scan(&tCount)
		if err != nil {
			return 0, d.HErr(err)
		}

		if lPars.OnlyCount {
			return tCount, nil
		}
	}

	dstV := reflect.ValueOf(dst)

	if dstV.Kind() != reflect.Pointer {
		return 0, d.HErr(errors.New("dst must be pointer to slice"))
	}

	dstV = reflect.Indirect(dstV)

	if dstV.Kind() != reflect.Slice {
		return 0, d.HErr(errors.New("dst must be pointer to slice"))
	}

	elemType := dstV.Type().Elem()

	if elemType.Kind() != reflect.Struct {
		return 0, d.HErr(errors.New("dst element type must struct"))
	}

	if dstV.IsNil() {
		dstV.Set(reflect.MakeSlice(reflect.SliceOf(elemType), 0, 10))
	}

	// generate columns
	colNames, colExps := d.HfGenerateColumns(lPars.Cols, allowedCols)

	scanItem := reflect.New(elemType).Elem()

	elemTypeFields := reflect.VisibleFields(elemType)

	scanFields := make([]any, len(colNames))

	for cnI, cn := range colNames {
		for _, field := range elemTypeFields {
			if field.Anonymous || !field.IsExported() {
				continue
			}

			fieldTag := field.Tag.Get(d.opts.FieldTag)
			if fieldTag == "" || strings.SplitN(fieldTag, ",", 2)[0] != cn {
				continue
			}

			scanFields[cnI] = scanItem.FieldByIndex(field.Index).Addr().Interface()

			break
		}
	}

	qOrderBy := ``

	if len(lPars.Sort) > 0 {
		if sortExprs := d.HfGenerateSort(lPars.Sort, allowedSorts); len(sortExprs) > 0 {
			qOrderBy = ` order by ` + strings.Join(sortExprs, ", ")
		}
	} else if lPars.SortName != "" {
		if sortExprs := allowedSortNames[lPars.SortName]; sortExprs != "" {
			qOrderBy = ` order by ` + sortExprs
		}
	}

	qOffset := ``
	qLimit := ``

	if lPars.PageSize > 0 {
		qOffset = ` offset ` + strconv.FormatInt(lPars.Page*lPars.PageSize, 10)
		qLimit = ` limit ` + strconv.FormatInt(lPars.PageSize, 10)
	}

	query := `select ` + strings.Join(colExps, ",") +
		` from ` + strings.Join(tables, " ") +
		` where ` + strings.Join(conds, " and ") +
		qOrderBy +
		qOffset +
		qLimit

	rows, err := d.DbQueryM(ctx, query, args)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(scanFields...)
		if err != nil {
			return 0, d.HErr(err)
		}

		dstV.Set(reflect.Append(dstV, scanItem))
	}
	if err = rows.Err(); err != nil {
		return 0, d.HErr(err)
	}

	return tCount, nil
}

func (d *St) HfGenerateColumns(rNames []string, allowed map[string]string) ([]string, []string) {
	colNames := make([]string, 0, len(allowed))
	colExps := make([]string, 0, cap(colNames))

	var ok bool
	var cn, exp string

	if len(rNames) == 0 {
		for k, v := range allowed {
			colNames = append(colNames, k)
			colExps = append(colExps, v)
		}
	} else {
		for _, cn = range rNames {
			if exp, ok = allowed[cn]; ok {
				colNames = append(colNames, cn)
				colExps = append(colExps, exp)
			}
		}
	}

	return colNames, colExps
}

func (d *St) HfGenerateSort(rNames []string, allowed map[string]string) []string {
	var expr string

	if len(rNames) == 0 {
		if expr = allowed["default"]; expr != "" {
			return []string{expr}
		}
		return []string{}
	}

	res := make([]string, 0, len(allowed))

	for _, sn := range rNames {
		if expr = allowed[sn]; expr != "" {
			res = append(res, expr)
		}
	}

	return res
}

func (d *St) HfGet(ctx context.Context, dst any, tables, conds []string, args map[string]any, allowedCols map[string]string) error {
	dstV := reflect.ValueOf(dst)

	if dstV.Kind() != reflect.Pointer {
		return d.HErr(errors.New("dst must be pointer to slice"))
	}

	dstV = reflect.Indirect(dstV)

	// temporary
	if dstV.Kind() != reflect.Struct {
		return d.HErr(errors.New("dst element type must struct"))
	}

	colNames := make([]string, 0, len(allowedCols))
	colExps := make([]string, 0, len(allowedCols))

	for cn, expr := range allowedCols {
		colNames = append(colNames, cn)
		colExps = append(colExps, expr)
	}

	elemTypeFields := reflect.VisibleFields(dstV.Type())

	scanFields := make([]any, len(colNames))

	for cnI, cn := range colNames {
		for _, field := range elemTypeFields {
			if field.Anonymous || !field.IsExported() {
				continue
			}

			fieldTag := field.Tag.Get(d.opts.FieldTag)
			if fieldTag == "" || strings.SplitN(fieldTag, ",", 2)[0] != cn {
				continue
			}

			scanFields[cnI] = dstV.FieldByIndex(field.Index).Addr().Interface()

			break
		}
	}

	query := `select ` + strings.Join(colExps, ",") +
		` from ` + strings.Join(tables, " ") +
		` where ` + strings.Join(conds, " and ") +
		` limit 1`

	err := d.DbQueryRowM(ctx, query, args).Scan(scanFields...)
	if err != nil {
		return err
	}

	return nil
}

func (d *St) HfCreate(ctx context.Context, table string, obj any, retCol string, retV any) error {
	fMap := d.HfGetCUFields(obj)

	fields := make([]string, len(fMap))
	values := make([]string, len(fields))
	args := make([]any, len(fields))
	argCnt := 0

	for k, v := range fMap {
		fields[argCnt] = k
		values[argCnt] = "$" + strconv.Itoa(argCnt+1)
		args[argCnt] = v
		argCnt++
	}

	query := `
		insert into ` + table + `(` + strings.Join(fields, ",") + `)
        values (` + strings.Join(values, ",") + `)
	`

	if retCol != "" && retV != nil {
		return d.DbQueryRow(ctx, query+" returning "+retCol, args...).Scan(retV)
	} else {
		return d.DbExec(ctx, query, args...)
	}
}

func (d *St) HfGetCUFields(obj any) map[string]any {
	v := reflect.Indirect(reflect.ValueOf(obj))
	vt := v.Type()

	var vField reflect.Value
	var vtField reflect.StructField
	var fieldTag string

	result := make(map[string]any)

	for i := 0; i < v.NumField(); i++ {
		vField = v.Field(i)
		vtField = vt.Field(i)

		switch vtField.Type.Kind() {
		case reflect.Pointer, reflect.Slice:
		default:
			continue
		}

		fieldTag = vtField.Tag.Get(d.opts.FieldTag)
		if fieldTag == "" || fieldTag == "-" {
			continue
		}

		if vField.IsNil() {
			continue
		}

		if vtField.Tag.Get(d.opts.IgnoreFlagFieldTag) == "-" {
			continue
		}

		result[strings.SplitN(fieldTag, ",", 2)[0]] = vField.Interface()
	}

	return result
}
