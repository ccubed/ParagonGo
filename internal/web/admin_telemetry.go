package web

import (
	"net/http"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/telemetry"
)

func adminTelemetry(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "telemetry.html", nil)
}

func adminTelemetryAPI(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "telemetry-api.html", nil)
}

// GET /admin/api/v1/telemetry
// Query params: category, itemId, mobId, roomId, zone, date, dateFrom, dateTo, sort (asc|desc)
func apiV1GetTelemetry(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	qb := telemetry.Query()

	if v := q.Get("category"); v != "" {
		qb = qb.Category(v)
	}
	if v := q.Get("zone"); v != "" {
		qb = qb.Zone(v)
	}
	if v := q.Get("date"); v != "" {
		qb = qb.Date(v)
	}
	if v := q.Get("dateFrom"); v != "" {
		qb = qb.DateFrom(v)
	}
	if v := q.Get("dateTo"); v != "" {
		qb = qb.DateTo(v)
	}
	if v := q.Get("itemId"); v != "" {
		if id, err := strconv.Atoi(v); err == nil {
			qb = qb.ItemId(id)
		}
	}
	if v := q.Get("mobId"); v != "" {
		if id, err := strconv.Atoi(v); err == nil {
			qb = qb.MobId(id)
		}
	}
	if v := q.Get("roomId"); v != "" {
		if id, err := strconv.Atoi(v); err == nil {
			qb = qb.RoomId(id)
		}
	}

	if q.Get("sort") == "asc" {
		qb = qb.SortAsc()
	} else {
		qb = qb.SortDesc()
	}

	results := qb.Results()
	if results == nil {
		results = []telemetry.Record{}
	}

	writeJSON(w, http.StatusOK, APIResponse[[]telemetry.Record]{
		Success: true,
		Data:    results,
	})
}

// DELETE /admin/api/v1/telemetry
// Query params: category, zone, date, dateFrom, dateTo (all optional - omit to clear all)
func apiV1DeleteTelemetry(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	category := q.Get("category")
	zone := q.Get("zone")
	date := q.Get("date")
	dateTo := q.Get("dateTo")

	// A bare "date" param sets both bounds to that single day.
	// dateFrom/dateTo allow clearing a range.
	if date != "" && dateTo == "" {
		dateTo = date
	}
	if v := q.Get("dateFrom"); v != "" && date == "" {
		date = v
	}

	telemetry.Clear(category, zone, date, dateTo, 0, 0, 0)

	if err := telemetry.Save(); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[struct{}]{Success: true})
}
