package telemetry

import "sort"

// QueryBuilder builds a filtered, optionally sorted query over telemetry records.
// All filter methods return the same pointer for chaining.
type QueryBuilder struct {
	category string
	zone     string
	dateFrom string // >= YYYYMMDD
	dateTo   string // <= YYYYMMDD
	itemId   int
	mobId    int
	roomId   int
	descSort bool
	hasSort  bool
}

func (q *QueryBuilder) Category(c string) *QueryBuilder { q.category = c; return q }
func (q *QueryBuilder) Zone(z string) *QueryBuilder     { q.zone = z; return q }
func (q *QueryBuilder) ItemId(id int) *QueryBuilder     { q.itemId = id; return q }
func (q *QueryBuilder) MobId(id int) *QueryBuilder      { q.mobId = id; return q }
func (q *QueryBuilder) RoomId(id int) *QueryBuilder     { q.roomId = id; return q }

// Date filters to an exact YYYYMMDD date.
func (q *QueryBuilder) Date(d string) *QueryBuilder { q.dateFrom = d; q.dateTo = d; return q }

// DateFrom filters to records on or after d (YYYYMMDD).
func (q *QueryBuilder) DateFrom(d string) *QueryBuilder { q.dateFrom = d; return q }

// DateTo filters to records on or before d (YYYYMMDD).
func (q *QueryBuilder) DateTo(d string) *QueryBuilder { q.dateTo = d; return q }

// SortAsc sorts results by Count ascending.
func (q *QueryBuilder) SortAsc() *QueryBuilder { q.descSort = false; q.hasSort = true; return q }

// SortDesc sorts results by Count descending.
func (q *QueryBuilder) SortDesc() *QueryBuilder { q.descSort = true; q.hasSort = true; return q }

// Results returns a copy of all records matching the configured filters.
func (q *QueryBuilder) Results() []Record {
	out := make([]Record, 0)
	for _, r := range records {
		if !matchesFilter(r, q.category, q.zone, q.dateFrom, q.dateTo, q.itemId, q.mobId, q.roomId) {
			continue
		}
		out = append(out, r)
	}

	if q.hasSort {
		sort.Slice(out, func(i, j int) bool {
			if q.descSort {
				return out[i].Count > out[j].Count
			}
			return out[i].Count < out[j].Count
		})
	}

	return out
}

// Total returns the sum of Count across all matching records.
func (q *QueryBuilder) Total() int {
	total := 0
	for _, r := range records {
		if matchesFilter(r, q.category, q.zone, q.dateFrom, q.dateTo, q.itemId, q.mobId, q.roomId) {
			total += r.Count
		}
	}
	return total
}
