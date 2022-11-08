package tests

import (
	"context"
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/supernova0730/dop/adapters/db"
	"github.com/supernova0730/dop/adapters/db/pg"
	"github.com/supernova0730/dop/adapters/logger/zap"
	"github.com/supernova0730/dop/dopTools"
	"github.com/supernova0730/dop/dopTypes"
)

var (
	bgCtx = context.Background()
	app   = struct {
		lg *zap.St
		db *pg.St
	}{
		lg: zap.New("info", true),
	}
)

func errCheck(err error) {
	if err != nil {
		app.lg.Fatal(err)
	}
}

func TestMain(m *testing.M) {
	var err error

	viper.AutomaticEnv()

	viper.SetDefault("PG_DSN", "postgres://localhost/dop")

	app.db, err = pg.New(true, app.lg, pg.OptionsSt{
		Dsn: viper.GetString("PG_DSN"),
	})
	errCheck(err)

	os.Exit(m.Run())
}

func TestDbPgHfList(t *testing.T) {
	err := app.db.DbExec(bgCtx, `drop table if exists t1 cascade`)
	errCheck(err)

	err = app.db.DbExec(bgCtx, `
		create table t1 (
			c1 int,
			c2 int[],
			c3 jsonb,
			c4 text
		);
	`)
	errCheck(err)

	err = app.db.DbExec(bgCtx, `truncate t1 restart identity cascade`)
	errCheck(err)

	err = app.db.DbExec(bgCtx, `
		insert into t1 (c1, c2, c3, c4) values
			(1, array [1, 2, 3, 4], '{"a": 1, "b": [1, 2], "c": "asd"}', '123')
			, (2, array [4,3,1], '{"a": 4, "b": [8], "c": "iii"}', 'poi')
	`)
	errCheck(err)

	type T1C3St struct {
		A int64   `json:"a"`
		B []int64 `json:"b"`
		C string  `json:"c"`
	}

	type T1SubSt struct {
		C2 *[]int64 `json:"c2" db:"c2"`
		C3 *T1C3St  `json:"c3" db:"c3"`
		C4 *string  `json:"c4" db:"c4"`
	}

	type T1St struct {
		C1 *int64 `json:"c1" db:"c1"`
		CC string `json:"cc" db:"cc"`
		T1SubSt
	}

	result := make([]*T1St, 0, 10)

	_, err = app.db.HfList(bgCtx, db.RDBListOptions{
		Dst: &result,
		ColExprs: map[string]string{
			"cc": `'hello'`,
		},
		Tables: []string{`t1`},
		LPars:  dopTypes.ListParams{},
		AllowedSorts: map[string]string{
			"default": "c1",
		},
	})
	// _, err = db.HfList(
	// 	bgCtx,
	// 	&result,
	// 	[]string{`t1`},
	// 	nil,
	// 	nil,
	// 	dopTypes.ListParams{},
	// 	nil,
	// 	map[string]string{
	// 		"default": "c1",
	// 	},
	// 	nil,
	// )
	errCheck(err)

	require.Len(t, result, 2)

	require.Equal(t, &T1St{
		C1: dopTools.NewPtr(int64(1)),
		CC: "hello",
		T1SubSt: T1SubSt{
			C2: dopTools.NewSlicePtr(int64(1), int64(2), int64(3), int64(4)),
			C3: &T1C3St{
				A: 1,
				B: []int64{1, 2},
				C: "asd",
			},
			C4: dopTools.NewPtr("123"),
		},
	}, result[0])

	require.Equal(t, &T1St{
		C1: dopTools.NewPtr(int64(2)),
		CC: "hello",
		T1SubSt: T1SubSt{
			C2: dopTools.NewSlicePtr(int64(4), int64(3), int64(1)),
			C3: &T1C3St{
				A: 4,
				B: []int64{8},
				C: "iii",
			},
			C4: dopTools.NewPtr("poi"),
		},
	}, result[1])
}
