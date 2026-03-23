package response

// PaginationMeta represents pagination metadata
type PaginationMeta struct {
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Pages int `json:"pages"`
}

// PaginatedResponse wraps a response with pagination metadata
type PaginatedResponse struct {
	Success    bool            `json:"success"`
	Data       interface{}     `json:"data"`
	Pagination *PaginationMeta `json:"pagination,omitempty"`
}

// CalculatePages calculates the total number of pages
func CalculatePages(total, limit int) int {
	if limit <= 0 {
		return 0
	}
	pages := total / limit
	if total%limit != 0 {
		pages++
	}
	return pages
}

// NewPaginatedResponse creates a new paginated response
func NewPaginatedResponse(data interface{}, total, page, limit int) *PaginatedResponse {
	return &PaginatedResponse{
		Success: true,
		Data:    data,
		Pagination: &PaginationMeta{
			Total: total,
			Page:  page,
			Limit: limit,
			Pages: CalculatePages(total, limit),
		},
	}
}
