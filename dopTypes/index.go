package dopTypes

type ErrRep struct {
	ErrorCode string            `json:"error_code"`
	Desc      string            `json:"desc,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
}

type ListParams struct {
	Cols           []string `json:"cols" form:"cols"`
	Page           int64    `json:"page" form:"page"`
	PageSize       int64    `json:"page_size" form:"page_size"`
	WithTotalCount bool     `json:"with_total_count" form:"with_total_count"`
	OnlyCount      bool     `json:"only_count" form:"only_count"`
	SortName       string   `json:"sort_name" form:"sort_name"`
	Sort           []string `json:"sort" form:"sort"`
}

type PaginatedListRep struct {
	Page       int64 `json:"page"`
	PageSize   int64 `json:"page_size"`
	TotalCount int64 `json:"total_count"`

	// Results    interface{} `json:"results"`
}
