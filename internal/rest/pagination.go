package rest

// PaginationOutput is the shared pagination envelope returned by every list
// endpoint's response body, embedded alongside the `items` field.
type PaginationOutput struct {
	Limit      int32 `json:"limit"`
	Offset     int32 `json:"offset"`
	TotalCount int32 `json:"total_count"`
	HasMore    bool  `json:"has_more"`
}

// newPagination builds the pagination envelope. limit/offset are normalized
// by the service layer (default limit 50); totalCount comes from the query.
func newPagination(limit, offset, totalCount int32) PaginationOutput {
	return PaginationOutput{
		Limit:      limit,
		Offset:     offset,
		TotalCount: totalCount,
		HasMore:    offset+limit < totalCount,
	}
}
