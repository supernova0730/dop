package dopErrs

type Err string

func (e Err) Error() string {
	return string(e)
}

type ErrWithDesc struct {
	Err  Err
	Desc string
}

func (e ErrWithDesc) Error() string {
	return e.Err.Error() + ", desc:" + e.Desc
}

const (
	NoRows            = Err("err_no_rows")
	BadColumnName     = Err("bad_column_name")
	BadJson           = Err("bad_json")
	BadJwt            = Err("bad_json")
	BadQueryParams    = Err("bad_query_params")
	ServiceNA         = Err("server_not_available")
	NotAuthorized     = Err("not_authorized")
	PermissionDenied  = Err("permission_denied")
	ObjectNotFound    = Err("object_not_found")
	IncorrectPageSize = Err("incorrect_page_size")
	BadStatusCode     = Err("bad_status_code")
)
