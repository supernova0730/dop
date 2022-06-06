package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/rendau/dop/adapters/db/pg"
	"github.com/rendau/dop/adapters/logger/zap"
	"github.com/rendau/dop/dopTypes"
)

func BenchmarkDop(b *testing.B) {
	ctx := context.Background()

	lg := zap.New("info", true)

	db, err := pg.New(true, lg, pg.OptionsSt{
		Dsn: "postgres://localhost/dop",
	})
	if err != nil {
		b.Fatal(err)
	}

	type T1C3St struct {
		A int64   `json:"a"`
		B []int64 `json:"b"`
		C string  `json:"c"`
	}

	type T1SubSt struct {
		C2 *[]int64 `json:"c2"`
		C3 *T1C3St  `json:"c3"`
		C4 *string  `json:"c4"`
	}

	type T1St struct {
		C1 *int64 `json:"c1"`
		T1SubSt
	}

	b.Run("Multi item scan", func(b *testing.B) {
		result := make([]*T1St, 0, 10)

		for i := 0; i < b.N; i++ {
			result = result[:0]

			_, err = db.HfList1(
				ctx,
				&result,
				[]string{`t1`},
				[]string{},
				map[string]interface{}{},
				dopTypes.ListParams{},
				map[string]string{
					"c1": "c1",
					"c2": "c2",
					"c3": "c3",
					"c4": "c4",
				},
				map[string]string{
					"default": "c1",
				},
				nil,
			)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func TestDop(t *testing.T) {
	ctx := context.Background()

	lg := zap.New("info", true)

	db, err := pg.New(true, lg, pg.OptionsSt{
		Dsn: "postgres://localhost/dop",
	})
	if err != nil {
		t.Fatal(err)
	}

	type T1C3St struct {
		A int64   `json:"a"`
		B []int64 `json:"b"`
		C string  `json:"c"`
	}

	type T1SubSt struct {
		C2 *[]int64 `json:"c2"`
		C3 *T1C3St  `json:"c3"`
		C4 *string  `json:"c4"`
	}

	type T1St struct {
		C1 *int64 `json:"c1"`
		T1SubSt
	}

	result := make([]*T1St, 0, 10)

	_, err = db.HfList(
		ctx,
		&result,
		[]string{`t1`},
		[]string{},
		map[string]interface{}{},
		dopTypes.ListParams{},
		map[string]string{
			"c1": "c1",
			"c2": "c2",
			"c3": "c3",
			"c4": "c4",
		},
		map[string]string{
			"default": "c1",
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	for _, item := range result {
		if item.C1 == nil {
			fmt.Printf("C1: nil")
		} else {
			fmt.Printf("C1: %v", *item.C1)
		}

		if item.C2 == nil {
			fmt.Printf("\t\t\t\t\tC2: nil")
		} else {
			fmt.Printf("\t\t\t\t\tC2: %v", *item.C2)
		}

		if item.C3 == nil {
			fmt.Printf("\t\t\t\t\tC3: nil")
		} else {
			fmt.Printf("\t\t\t\t\tC3: %v", *item.C3)
		}

		if item.C4 == nil {
			fmt.Printf("\t\t\t\t\tC4: nil")
		} else {
			fmt.Printf("\t\t\t\t\tC4: %v", *item.C4)
		}

		fmt.Print("\n")
	}
}
