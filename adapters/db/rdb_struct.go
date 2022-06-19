package db

import (
	"github.com/rendau/dop/dopTypes"
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
