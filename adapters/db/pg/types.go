package pg

import (
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// Options

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

func (o OptionsSt) getConfig() (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig(o.Dsn)
	if err != nil {
		return nil, err
	}

	// default values
	cfg.ConnConfig.RuntimeParams["timezone"] = defaultOptions.Timezone
	cfg.MaxConns = defaultOptions.MaxConns
	cfg.MinConns = defaultOptions.MinConns
	cfg.MaxConnLifetime = defaultOptions.MaxConnLifetime
	cfg.MaxConnIdleTime = defaultOptions.MaxConnIdleTime
	cfg.HealthCheckPeriod = defaultOptions.HealthCheckPeriod
	cfg.LazyConnect = defaultOptions.LazyConnect

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
