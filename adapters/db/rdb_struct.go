package db

import (
	"github.com/supernova0730/dop/dopTypes"
)

type RDBListOptions struct {
	Dst              any
	Tables           []string
	LPars            dopTypes.ListParams
	Conds            []string
	Args             map[string]any
	ColExprs         map[string]string
	AllowedSorts     map[string]string
	AllowedSortNames map[string]string
}

type RDBGetOptions struct {
	Dst      any
	Tables   []string
	Conds    []string
	Args     map[string]any
	ColExprs map[string]string
}

type RDBCreateOptions struct {
	Table  string
	Obj    any
	RetCol string
	RetV   any
}

type RDBUpdateOptions struct {
	Table string
	Obj   any
	Conds []string
	Args  map[string]any
}

type RDBDeleteOptions struct {
	Table string
	Conds []string
	Args  map[string]any
}
