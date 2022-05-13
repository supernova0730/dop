package pg

import (
	"time"

	"github.com/jackc/pgx/v4"
)

// Options

type OptionsSt struct {
	Dsn                string
	Timezone           string
	MaxConns           int32
	MinConns           int32
	MaxConnLifetime    time.Duration
	MaxConnIdleTime    time.Duration
	HealthCheckPeriod  time.Duration
	FieldTag           string
	IgnoreFlagFieldTag string
}

func (o *OptionsSt) mergeWithDefaults() {
	if o.Timezone == "" {
		o.Timezone = defaultOptions.Timezone
	}
	if o.MaxConns == 0 {
		o.MaxConns = defaultOptions.MaxConns
	}
	if o.MinConns == 0 {
		o.MinConns = defaultOptions.MinConns
	}
	if o.MaxConnLifetime == 0 {
		o.MaxConnLifetime = defaultOptions.MaxConnLifetime
	}
	if o.MaxConnIdleTime == 0 {
		o.MaxConnIdleTime = defaultOptions.MaxConnIdleTime
	}
	if o.HealthCheckPeriod == 0 {
		o.HealthCheckPeriod = defaultOptions.HealthCheckPeriod
	}
	if o.FieldTag == "" {
		o.FieldTag = defaultOptions.FieldTag
	}
	if o.IgnoreFlagFieldTag == "" {
		o.IgnoreFlagFieldTag = defaultOptions.IgnoreFlagFieldTag
	}
}

type txContainerSt struct {
	tx pgx.Tx
}

type rowsSt struct {
	pgx.Rows
	db Connection
}

func (o rowsSt) Err() error {
	return o.db.HErr(o.Rows.Err())
}

func (o rowsSt) Scan(dest ...interface{}) error {
	return o.db.HErr(o.Rows.Scan(dest...))
}

type rowSt struct {
	pgx.Row
	db Connection
}

func (o rowSt) Scan(dest ...interface{}) error {
	return o.db.HErr(o.Row.Scan(dest...))
}
