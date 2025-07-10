// internal/repository/pagination.go
package repository

// Pagination holds standard offset/limit paging parameters.
type Pagination struct {
	Page     int // 1-based page number
	PageSize int // number of items per page
}

// Offset returns the number of records to skip (0-based).
func (p Pagination) Offset() int {
	if p.Page <= 1 {
		return 0
	}
	return (p.Page - 1) * p.PageSize
}

// Limit returns the maximum number of records to return.
func (p Pagination) Limit() int {
	if p.PageSize <= 0 {
		return 10 // a sensible default
	}
	return p.PageSize
}
