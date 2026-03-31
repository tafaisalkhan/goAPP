package country

type Country struct {
	ID          int64  `json:"id"`
	CountryCode string `json:"countrycode"`
	CountryName string `json:"countryname"`
	Code        string `json:"code"`
}

type UpdatePatch map[string]string

type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}

type ListResponse struct {
	Countries  []Country  `json:"countries"`
	Pagination Pagination `json:"pagination"`
}
