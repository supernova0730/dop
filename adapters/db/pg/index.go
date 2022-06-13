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
	"github.com/rendau/dop/adapters/db"
	"github.com/rendau/dop/adapters/logger"
	"github.com/rendau/dop/dopErrs"
	"github.com/rendau/dop/dopTypes"
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

func (d *St) getCon(ctx context.Context) db.RDBConSt {
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

func (d *St) TransactionFn(ctx context.Context, f func(context.Context) error) error {
	var err error

	if ctx == nil {
		ctx = context.Background()
	}

	if ctx, err = d.ContextWithTransaction(ctx); err != nil {
		return err
	}
	defer func() { d.RollbackContextTransaction(ctx) }()

	err = f(ctx)
	if err != nil {
		return err
	}

	return d.CommitContextTransaction(ctx)
}

// query

func (d *St) DbExec(ctx context.Context, sql string, args ...any) error {
	_, err := d.getCon(ctx).Exec(ctx, sql, args...)
	return d.HErr(err)
}

func (d *St) DbQuery(ctx context.Context, sql string, args ...any) (db.RDBRows, error) {
	rows, err := d.getCon(ctx).Query(ctx, sql, args...)
	return rowsSt{Rows: rows, db: d}, d.HErr(err)
}

func (d *St) DbQueryRow(ctx context.Context, sql string, args ...any) db.RDBRow {
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

func (d *St) DbQueryM(ctx context.Context, sql string, argMap map[string]any) (db.RDBRows, error) {
	rbSql, args := d.queryRebindNamed(sql, argMap)
	rows, err := d.getCon(ctx).Query(ctx, rbSql, args...)
	return rowsSt{Rows: rows, db: d}, d.HErr(err)
}

func (d *St) DbQueryRowM(ctx context.Context, sql string, argMap map[string]any) db.RDBRow {
	rbSql, args := d.queryRebindNamed(sql, argMap)
	return rowSt{Row: d.getCon(ctx).QueryRow(ctx, rbSql, args...), db: d}
}

func (d *St) HErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, pgx.ErrNoRows), errors.Is(err, sql.ErrNoRows):
		err = dopErrs.NoRows
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
	lPars dopTypes.ListParams,
	allowedCols map[string]string,
	allowedSorts map[string]string,
	allowedSortNames map[string]string,
) (int64, error) {
	var tCount int64

	qWhere := d.HfOptionalWhere(conds)

	if (lPars.WithTotalCount && lPars.PageSize > 0) || lPars.OnlyCount {
		err := d.DbQueryRowM(ctx, `select count(*)`+
			` from `+strings.Join(tables, " ")+
			qWhere, args).Scan(&tCount)
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

	elemBaseType := dstV.Type().Elem()

	elemType := elemBaseType

	elemIsPtr := false

	if elemType.Kind() == reflect.Pointer {
		elemType = elemType.Elem()
		elemIsPtr = true
	}

	if elemType.Kind() != reflect.Struct {
		return 0, d.HErr(errors.New("dst element type must struct"))
	}

	if dstV.IsNil() {
		dstV.Set(reflect.MakeSlice(reflect.SliceOf(elemBaseType), 0, 10))
	}

	elemFieldNameMap := d.hfGetStructFieldMap(reflect.VisibleFields(elemType))

	// generate columns
	colNames, colExps := d.hfGenerateColumns(lPars.Cols, allowedCols, elemFieldNameMap)

	scanFieldNames := make([]string, 0, len(colNames))

	var fieldName string

	for _, cn := range colNames {
		if fieldName = elemFieldNameMap[cn]; fieldName != "" {
			scanFieldNames = append(scanFieldNames, fieldName)
		} else {
			return 0, d.HErr(errors.New("field '" + cn + "' not found in element struct"))
		}
	}

	qOrderBy := ``

	if lPars.SortName != "" {
		if sortExprs := allowedSortNames[lPars.SortName]; sortExprs != "" {
			qOrderBy = ` order by ` + sortExprs
		}
	} else {
		if sortExprs := d.HfGenerateSort(lPars.Sort, allowedSorts); len(sortExprs) > 0 {
			qOrderBy = ` order by ` + strings.Join(sortExprs, ", ")
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
		qWhere +
		qOrderBy +
		qOffset +
		qLimit

	// fmt.Println(query)

	rows, err := d.DbQueryM(ctx, query, args)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var scanItemPtr reflect.Value
	var scanItem reflect.Value
	scanFields := make([]any, len(scanFieldNames))

	for rows.Next() {
		scanItemPtr = reflect.New(elemType)
		scanItem = scanItemPtr.Elem()

		for i, fName := range scanFieldNames {
			scanFields[i] = scanItem.FieldByName(fName).Addr().Interface()
		}

		err = rows.Scan(scanFields...)
		if err != nil {
			return 0, d.HErr(err)
		}

		if elemIsPtr {
			dstV.Set(reflect.Append(dstV, scanItemPtr))
		} else {
			dstV.Set(reflect.Append(dstV, scanItem))
		}
	}
	if err = rows.Err(); err != nil {
		return 0, d.HErr(err)
	}

	return tCount, nil
}

func (d *St) HfGenerateColumns(rNames []string, allowed map[string]string) ([]string, []string) {
	return d.hfGenerateColumns(rNames, allowed, nil)
}

func (d *St) hfGenerateColumns(rNames []string, allowed map[string]string, structFieldNameMap map[string]string) ([]string, []string) {
	if len(allowed) == 0 {
		allowed = make(map[string]string, len(structFieldNameMap))

		for tagName := range structFieldNameMap {
			allowed[tagName] = tagName
		}
	}

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

	if dstV.Kind() != reflect.Struct {
		return d.HErr(errors.New("dst element type must struct"))
	}

	elemFieldNameMap := d.hfGetStructFieldMap(reflect.VisibleFields(dstV.Type()))

	if len(allowedCols) == 0 {
		allowedCols = make(map[string]string, len(elemFieldNameMap))

		for tagName := range elemFieldNameMap {
			allowedCols[tagName] = tagName
		}
	}

	colNames := make([]string, 0, len(allowedCols))
	colExps := make([]string, 0, len(allowedCols))

	for cn, expr := range allowedCols {
		colNames = append(colNames, cn)
		colExps = append(colExps, expr)
	}

	scanFields := make([]any, len(colNames))

	var fieldName string

	for cnI, cn := range colNames {
		if fieldName = elemFieldNameMap[cn]; fieldName != "" {
			scanFields[cnI] = dstV.FieldByName(fieldName).Addr().Interface()
		} else {
			return d.HErr(errors.New("field '" + cn + "' not found in element struct"))
		}
	}

	query := `select ` + strings.Join(colExps, ",") +
		` from ` + strings.Join(tables, " ") +
		d.HfOptionalWhere(conds) +
		` limit 1`

	err := d.DbQueryRowM(ctx, query, args).Scan(scanFields...)
	if err != nil {
		// if nilOnNoRows && errors.Is(err, dopErrs.NoRows) {
		// 	return nil
		// }
		return err
	}

	return nil
}

func (d *St) hfGetStructFieldMap(fields []reflect.StructField) map[string]string {
	result := make(map[string]string, 30)

	for _, field := range fields {
		if field.Anonymous || !field.IsExported() {
			continue
		}

		fieldTag := field.Tag.Get(d.opts.FieldTag)
		if fieldTag != "" {
			fieldTag = strings.SplitN(fieldTag, ",", 2)[0]
		}
		if fieldTag == "" || fieldTag == "-" {
			continue
		}

		result[fieldTag] = field.Name
	}

	return result
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

func (d *St) HfUpdate(ctx context.Context, table string, obj any, conds []string, condArgs map[string]any) error {
	fMap := d.HfGetCUFields(obj)

	fields := make([]string, 0, len(fMap))

	for k := range fMap {
		fields = append(fields, k+`=${`+k+`}`)
	}

	if len(fields) == 0 {
		return nil
	}

	query := `
		update ` + table + `
		set ` + strings.Join(fields, ",")

	if len(conds) > 0 {
		query += ` where ` + strings.Join(conds, " and ")

		for k, v := range condArgs {
			fMap[k] = v
		}
	}

	return d.DbExecM(ctx, query, fMap)
}

func (d *St) HfGetCUFields(obj any) map[string]any {
	v := reflect.Indirect(reflect.ValueOf(obj))

	vFields := reflect.VisibleFields(v.Type())

	result := make(map[string]any, len(vFields))

	var vField reflect.Value
	var fieldTag []string

	for _, field := range vFields {
		switch field.Type.Kind() {
		case reflect.Pointer, reflect.Slice:
		default:
			continue
		}

		fieldTag = strings.Split(field.Tag.Get(d.opts.FieldTag), ",")
		if len(fieldTag) == 0 || fieldTag[0] == "-" {
			continue
		}

		vField = v.FieldByIndex(field.Index)

		if vField.IsNil() {
			continue
		}

		result[fieldTag[0]] = vField.Interface()
	}

	return result
}

func (d *St) HfOptionalWhere(conds []string) string {
	if len(conds) > 0 {
		return ` where ` + strings.Join(conds, " and ") + ` `
	}
	return ``
}

func (d *St) HfDelete(ctx context.Context, table string, conds []string, args map[string]any) error {
	return d.DbExecM(ctx, `delete from `+table+d.HfOptionalWhere(conds), args)
}
