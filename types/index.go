package types

type ListParams struct {
	Cols           []string `json:"cols"`
	Page           int64    `json:"page"`
	PageSize       int64    `json:"page_size"`
	WithTotalCount bool     `json:"with_total_count"`
	OnlyCount      bool     `json:"only_count"`
	SortName       string   `json:"sort_name"`
	Sort           []string `json:"sort"`
}
