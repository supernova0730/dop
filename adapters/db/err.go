package db

type Err string

func (e Err) Error() string {
	return string(e)
}

const (
	ErrNoRows = Err("err_no_rows")
)
