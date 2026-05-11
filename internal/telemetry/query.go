package telemetry

import (
	"fmt"
	"sort"
)

// GroupByField names the dimension used by GroupBy.
type GroupByField uint8

const (
	GroupByMobId GroupByField = iota
	GroupByItemId
	GroupByZone
	GroupByRoomId
	GroupByDate
	GroupByCategory
	GroupByRaceId
	GroupByTopic
)

// QueryBuilder builds a filtered, optionally sorted query over telemetry records.
// All filter methods return the same pointer for chaining.
type QueryBuilder struct {
	category   string
	zone       string
	dateFrom   string // >= YYYYMMDD
	dateTo     string // <= YYYYMMDD
	itemId     int
	mobId      int
	roomId     int
	raceId     int
	topic      string
	descSort   bool
	hasSort    bool
	groupBy    GroupByField
	hasGroupBy bool
}

func (q *QueryBuilder) Category(c string) *QueryBuilder { q.category = c; return q }
func (q *QueryBuilder) Zone(z string) *QueryBuilder     { q.zone = z; return q }
func (q *QueryBuilder) ItemId(id int) *QueryBuilder     { q.itemId = id; return q }
func (q *QueryBuilder) MobId(id int) *QueryBuilder      { q.mobId = id; return q }
func (q *QueryBuilder) RoomId(id int) *QueryBuilder     { q.roomId = id; return q }
func (q *QueryBuilder) RaceId(id int) *QueryBuilder     { q.raceId = id; return q }
func (q *QueryBuilder) Topic(t string) *QueryBuilder    { q.topic = t; return q }

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

// GroupBy collapses all matching records into one row per unique value of the
// given field, summing their counts. Date, Zone, RoomId, and other fields on
// the rolled-up record are left blank/zero because they are no longer
// meaningful across multiple source records.
func (q *QueryBuilder) GroupBy(field GroupByField) *QueryBuilder {
	q.groupBy = field
	q.hasGroupBy = true
	return q
}

// Results returns a copy of all records matching the configured filters.
// If GroupBy was called, records are rolled up by that field before sorting.
func (q *QueryBuilder) Results() []Record {
	out := make([]Record, 0)
	for _, r := range records {
		if !matchesFilter(r, q.category, q.zone, q.dateFrom, q.dateTo, q.itemId, q.mobId, q.roomId, q.raceId, q.topic) {
			continue
		}
		out = append(out, r)
	}

	if q.hasGroupBy {
		out = rollup(out, q.groupBy)
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
		if matchesFilter(r, q.category, q.zone, q.dateFrom, q.dateTo, q.itemId, q.mobId, q.roomId, q.raceId, q.topic) {
			total += r.Count
		}
	}
	return total
}

// rollup merges src into one record per unique value of field, summing counts.
func rollup(src []Record, field GroupByField) []Record {
	type bucket struct {
		rec   Record
		count int
	}
	order := []string{}
	buckets := map[string]*bucket{}

	for _, r := range src {
		key := rollupKey(r, field)
		if b, ok := buckets[key]; ok {
			b.count += r.Count
		} else {
			proto := Record{Category: r.Category}
			switch field {
			case GroupByMobId:
				proto.MobId = r.MobId
			case GroupByItemId:
				proto.ItemId = r.ItemId
			case GroupByZone:
				proto.Zone = r.Zone
			case GroupByRoomId:
				proto.RoomId = r.RoomId
			case GroupByDate:
				proto.Date = r.Date
			case GroupByCategory:
				proto.Category = r.Category
			case GroupByRaceId:
				proto.RaceId = r.RaceId
			case GroupByTopic:
				proto.Topic = r.Topic
			}
			buckets[key] = &bucket{rec: proto, count: r.Count}
			order = append(order, key)
		}
	}

	out := make([]Record, 0, len(buckets))
	for _, key := range order {
		b := buckets[key]
		b.rec.Count = b.count
		out = append(out, b.rec)
	}
	return out
}

func rollupKey(r Record, field GroupByField) string {
	switch field {
	case GroupByMobId:
		return fmt.Sprintf("%d", r.MobId)
	case GroupByItemId:
		return fmt.Sprintf("%d", r.ItemId)
	case GroupByZone:
		return r.Zone
	case GroupByRoomId:
		return fmt.Sprintf("%d", r.RoomId)
	case GroupByDate:
		return r.Date
	case GroupByCategory:
		return r.Category
	case GroupByRaceId:
		return fmt.Sprintf("%d", r.RaceId)
	case GroupByTopic:
		return r.Topic
	}
	return ""
}
