package rest

// PaginationOutput is the shared pagination envelope returned by every list
// endpoint's response body, embedded alongside the `items` field.
type PaginationOutput struct {
	Limit      int32 `json:"limit"`
	Offset     int32 `json:"offset"`
	TotalCount int32 `json:"total_count"`
	HasMore    bool  `json:"has_more"`
}

// newPagination builds the pagination envelope, normalizing limit/offset the
// same way the service layer does (default limit 50, offset floor 0) so the
// echoed values match what was actually applied to the query.
func newPagination(limit, offset, totalCount int32) PaginationOutput {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return PaginationOutput{
		Limit:      limit,
		Offset:     offset,
		TotalCount: totalCount,
		HasMore:    offset+limit < totalCount,
	}
}
