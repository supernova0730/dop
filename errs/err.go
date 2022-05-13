package errs

type Err string

func (e Err) Error() string {
	return string(e)
}

const (
	NoRows            = Err("err_no_rows")
	BadColumnName     = Err("bad_column_name")
	BadJson           = Err("bad_json")
	ServiceNA         = Err("server_not_available")
	NotAuthorized     = Err("not_authorized")
	PermissionDenied  = Err("permission_denied")
	ObjectNotFound    = Err("object_not_found")
	IncorrectPageSize = Err("incorrect_page_size")
)
