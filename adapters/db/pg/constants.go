package pg

import (
	"regexp"
	"time"
)

const (
	ErrPrefix         = "pg-error"
	TransactionCtxKey = "pg_transaction"
)

var defaultOptions = OptionsSt{
	Timezone:          "Asia/Almaty",
	MaxConns:          100,
	MinConns:          5,
	MaxConnLifetime:   30 * time.Minute,
	MaxConnIdleTime:   15 * time.Minute,
	HealthCheckPeriod: 20 * time.Second,
	FieldTag:          "db",
}

var (
	queryParamRegexp = regexp.MustCompile(`(?si)\$\{[^}]+\}`)
)
