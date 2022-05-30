package dopTypes

type ErrRep struct {
	// Код ошибки
	ErrorCode string `json:"error_code"`
}

type ListParams struct {
	Cols           []string `json:"cols"`
	Page           int64    `json:"page"`
	PageSize       int64    `json:"page_size"`
	WithTotalCount bool     `json:"with_total_count"`
	OnlyCount      bool     `json:"only_count"`
	SortName       string   `json:"sort_name"`
	Sort           []string `json:"sort"`
}

type PaginatedListRep struct {
	Page       int64       `json:"page"`
	PageSize   int64       `json:"page_size"`
	TotalCount int64       `json:"total_count"`
	Results    interface{} `json:"results"`
}
