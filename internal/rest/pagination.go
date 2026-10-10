package rest

import (
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

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

// CursorPaginationOutput is the pagination envelope of the high-volume
// newest-first lists (deliveries, event occurrences). There is no total:
// counting every match cost more than the page itself on large tables.
type CursorPaginationOutput struct {
	Limit      int32  `json:"limit"`
	HasMore    bool   `json:"has_more" doc:"True when older items follow this page."`
	NextCursor string `json:"next_cursor,omitempty" doc:"Pass as cursor to get the next, older page. Absent on the last page."`
}

// encodeCursor makes the opaque cursor for the row a page ended on.
// Microseconds match Postgres's timestamptz precision exactly.
func encodeCursor(createdAt time.Time, id uuid.UUID) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(createdAt.UnixMicro(), 10) + "." + id.String()))
}

// errOffsetRemoved rejects offset paging on the delivery and event
// occurrence lists. Those tables grow without bound and an offset reads and
// discards every skipped row (91ms for deliveries and 241ms for events at
// 100k deep on 6M rows, growing with depth), while the cursor stays under a
// millisecond at any depth.
var errOffsetRemoved = huma.Error400BadRequest("offset paging was removed from this list; pass pagination.next_cursor from the previous page as cursor")

// decodeCursor parses a cursor from encodeCursor; empty means none.
func decodeCursor(s string) (*store.PageCursor, error) {
	if s == "" {
		return nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, huma.Error400BadRequest("cursor is not valid: pass next_cursor from a previous page unchanged")
	}
	micros, idStr, ok := strings.Cut(string(raw), ".")
	us, errT := strconv.ParseInt(micros, 10, 64)
	id, errID := uuid.Parse(idStr)
	if !ok || errT != nil || errID != nil {
		return nil, huma.Error400BadRequest("cursor is not valid: pass next_cursor from a previous page unchanged")
	}
	return &store.PageCursor{CreatedAt: time.UnixMicro(us).UTC(), ID: id}, nil
}

// newCursorPagination builds the envelope for a page whose last row is
// (lastAt, lastID); the cursor is only set when more rows follow.
func newCursorPagination(limit int32, hasMore bool, lastAt time.Time, lastID uuid.UUID) CursorPaginationOutput {
	if limit <= 0 {
		limit = 50
	}
	out := CursorPaginationOutput{Limit: limit, HasMore: hasMore}
	if hasMore {
		out.NextCursor = encodeCursor(lastAt, lastID)
	}
	return out
}
