# telemetry — usage examples

## Querying in Go code

The `Query()` builder is the primary interface. All filter methods are chainable
and return the same `*QueryBuilder`. Call `Results()` or `Total()` to execute.

### All drops of a specific item, sorted by count descending

```go
drops := telemetry.Query().
    Category(telemetry.CatItemDrop).
    ItemId(42).
    SortDesc().
    Results()

for _, r := range drops {
    // r.MobId  — which mob dropped it (0 = floor/player drop)
    // r.Zone   — which zone
    // r.RoomId — which room
    // r.Date   — "YYYYMMDD"
    // r.Count  — how many times
    fmt.Printf("mob %d in %s: %d drops\n", r.MobId, r.Zone, r.Count)
}
```

### All items dropped by a specific mob

```go
drops := telemetry.Query().
    Category(telemetry.CatItemDrop).
    MobId(7).
    SortDesc().
    Results()
```

### Total mob kills in a zone this month

```go
// Build a YYYYMM prefix and use DateFrom/DateTo
now := time.Now()
firstOfMonth := fmt.Sprintf("%d%02d01", now.Year(), now.Month())
lastOfMonth  := fmt.Sprintf("%d%02d31", now.Year(), now.Month())

total := telemetry.Query().
    Category(telemetry.CatMobKill).
    Zone("frostfang").
    DateFrom(firstOfMonth).
    DateTo(lastOfMonth).
    Total()

fmt.Printf("frostfang mob kills this month: %d\n", total)
```

### Most-purchased items overall

```go
purchases := telemetry.Query().
    Category(telemetry.CatItemPurchase).
    SortDesc().
    Results()

// purchases[0] is the most-purchased item across all dates
```

### Player deaths by mob, filtered to a single day

```go
today := time.Now().Format("20060102")

deaths := telemetry.Query().
    Category(telemetry.CatPlayerDeath).
    Date(today).
    SortDesc().
    Results()
```

### Check whether a specific mob has ever dropped a specific item

```go
count := telemetry.Query().
    Category(telemetry.CatItemDrop).
    ItemId(42).
    MobId(7).
    Total()

if count > 0 {
    fmt.Printf("mob 7 has dropped item 42 %d times\n", count)
}
```

---

## API endpoint

`GET /admin/api/v1/telemetry`

All parameters are optional. Omit a parameter to match any value.

| Parameter  | Type   | Description                              |
|------------|--------|------------------------------------------|
| `category` | string | `item_drop`, `item_pickup`, `mob_kill`, `player_death`, `item_purchase` |
| `itemId`   | int    | Filter by item spec ID                   |
| `mobId`    | int    | Filter by mob spec ID                    |
| `roomId`   | int    | Filter by room ID                        |
| `zone`     | string | Filter by zone name (exact match)        |
| `date`     | string | Exact date `YYYYMMDD`                    |
| `dateFrom` | string | Start of date range `YYYYMMDD` inclusive |
| `dateTo`   | string | End of date range `YYYYMMDD` inclusive   |
| `sort`     | string | `asc` or `desc` (default `desc`)         |

`DELETE /admin/api/v1/telemetry`

Clears records matching the supplied filters and saves to disk. Accepts the
same filter parameters as GET except `sort`. Omit all parameters to clear
everything.

---

## Date range notes

`DateFrom` and `DateTo` use simple string comparison on `YYYYMMDD` format,
which sorts lexicographically the same as chronologically. Both bounds are
inclusive. Use `Date(d)` as a shorthand when you want exactly one day
(`DateFrom(d).DateTo(d)`).
