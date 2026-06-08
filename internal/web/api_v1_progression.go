package web

import (
	"math"
	"net/http"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

type progressionPreviewData struct {
	Levels       []int            `json:"levels"`
	StatGains    map[string][]int `json:"stat_gains"`
	StatGainsAdj map[string][]int `json:"stat_gains_adj"`
	HP           map[string][]int `json:"hp"`
	HPRaw        map[string][]int `json:"hp_raw"`
	Mana         map[string][]int `json:"mana"`
	ManaRaw      map[string][]int `json:"mana_raw"`
	XPPerLevel   []int            `json:"xp_per_level"`
	XPCumulative []int            `json:"xp_cumulative"`
}

// GET /admin/api/v1/progression/preview
//
// Accepts optional query parameters matching every ProgressionConfig field.
// When a parameter is present it overrides the live config for this preview
// computation only — nothing is saved. Returns precomputed chart series so the
// admin page can render accurate graphs without duplicating the formula in JS.
func apiV1GetProgressionPreview(w http.ResponseWriter, r *http.Request) {
	cfg := configs.GetProgressionConfig()

	// Override config from query params when provided.
	q := r.URL.Query()
	if v := q.Get("BaseModFactor"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.BaseModFactor = configs.ConfigFloat(f)
		}
	}
	if v := q.Get("BaseModExponent"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.BaseModExponent = configs.ConfigFloat(f)
		}
	}
	if v := q.Get("NaturalGainsModFactor"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.NaturalGainsModFactor = configs.ConfigFloat(f)
		}
	}
	if v := q.Get("NaturalGainsExponent"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.NaturalGainsExponent = configs.ConfigFloat(f)
		}
	}
	if v := q.Get("HPBase"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.HPBase = configs.ConfigInt(i)
		}
	}
	if v := q.Get("HPPerLevel"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.HPPerLevel = configs.ConfigFloat(f)
		}
	}
	if v := q.Get("HPPerVitality"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.HPPerVitality = configs.ConfigFloat(f)
		}
	}
	if v := q.Get("ManaBase"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.ManaBase = configs.ConfigInt(i)
		}
	}
	if v := q.Get("ManaPerLevel"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.ManaPerLevel = configs.ConfigFloat(f)
		}
	}
	if v := q.Get("ManaPerMysticism"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.ManaPerMysticism = configs.ConfigFloat(f)
		}
	}
	if v := q.Get("XPBase"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.XPBase = configs.ConfigInt(i)
		}
	}
	if v := q.Get("XPLevelFactor"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.XPLevelFactor = configs.ConfigFloat(f)
		}
	}
	if v := q.Get("XPLevelPower"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.XPLevelPower = configs.ConfigFloat(f)
		}
	}
	if v := q.Get("MaxLevel"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.MaxLevel = configs.ConfigInt(i)
		}
	}
	if v := q.Get("StatCapThreshold"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.StatCapThreshold = configs.ConfigInt(i)
		}
	}
	if v := q.Get("StatCapAnchor"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.StatCapAnchor = configs.ConfigInt(i)
		}
	}
	if v := q.Get("StatCapExponent"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.StatCapExponent = configs.ConfigFloat(f)
		}
	}
	if v := q.Get("StatCapScale"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.StatCapScale = configs.ConfigFloat(f)
		}
	}
	if v := q.Get("StatCapExemptBonus"); v != "" {
		cfg.StatCapExemptBonus = configs.ConfigBool(v == "true" || v == "1")
	}

	cfg.Validate()

	maxLevel := int(cfg.MaxLevel)

	// Downsample to at most 100 points for chart rendering.
	// When MaxLevel > 100 we pick evenly-spaced levels so the chart stays readable.
	const maxPoints = 100
	var chartLevels []int
	if maxLevel <= maxPoints {
		chartLevels = make([]int, maxLevel)
		for i := range chartLevels {
			chartLevels[i] = i + 1
		}
	} else {
		chartLevels = make([]int, maxPoints)
		for i := range chartLevels {
			// Evenly space from level 1 to maxLevel inclusive.
			chartLevels[i] = 1 + int(math.Round(float64(i)*(float64(maxLevel-1)/float64(maxPoints-1))))
		}
	}
	n := len(chartLevels)
	levels := chartLevels

	// Stat gains: three representative racial base values plus their compressed (ValueAdj) counterparts.
	// base1 = weak race, base3 = average, base5 = strong.
	statBases := map[string]int{"base1": 1, "base3": 3, "base5": 5}
	statGains := make(map[string][]int, len(statBases))
	statGainsAdj := make(map[string][]int, len(statBases))
	for label, base := range statBases {
		series := make([]int, n)
		seriesAdj := make([]int, n)
		for i, lvl := range levels {
			v := gainsForLevelWithCfg(lvl, base, cfg)
			series[i] = v
			seriesAdj[i] = applyCapWithCfg(v, cfg)
		}
		statGains[label] = series
		statGainsAdj[label] = seriesAdj
	}

	// HP and Mana: three representative racial base values for Vitality/Mysticism.
	// Vitality and Mysticism use the same GainsForLevel formula as other stats.
	vitatBases := map[string]int{"base1": 1, "base3": 3, "base5": 5}
	hp := make(map[string][]int, len(vitatBases))
	hpRaw := make(map[string][]int, len(vitatBases))
	mana := make(map[string][]int, len(vitatBases))
	manaRaw := make(map[string][]int, len(vitatBases))
	for label, base := range vitatBases {
		hpSeries := make([]int, n)
		hpRawSeries := make([]int, n)
		manaSeries := make([]int, n)
		manaRawSeries := make([]int, n)
		for i, lvl := range levels {
			rawStat := gainsForLevelWithCfg(lvl, base, cfg)
			adjStat := applyCapWithCfg(rawStat, cfg)
			hpSeries[i] = int(cfg.HPBase) +
				int(float64(lvl)*float64(cfg.HPPerLevel)) +
				int(float64(adjStat)*float64(cfg.HPPerVitality))
			hpRawSeries[i] = int(cfg.HPBase) +
				int(float64(lvl)*float64(cfg.HPPerLevel)) +
				int(float64(rawStat)*float64(cfg.HPPerVitality))
			manaSeries[i] = int(cfg.ManaBase) +
				int(float64(lvl)*float64(cfg.ManaPerLevel)) +
				int(float64(adjStat)*float64(cfg.ManaPerMysticism))
			manaRawSeries[i] = int(cfg.ManaBase) +
				int(float64(lvl)*float64(cfg.ManaPerLevel)) +
				int(float64(rawStat)*float64(cfg.ManaPerMysticism))
		}
		hp[label] = hpSeries
		hpRaw[label] = hpRawSeries
		mana[label] = manaSeries
		manaRaw[label] = manaRawSeries
	}

	// XP curve (TNLScale = 1.0 for display purposes).
	xpPerLevel := make([]int, n)
	xpCumulative := make([]int, n)
	for i, lvl := range levels {
		xpForLevel := xpTLWithCfg(lvl, cfg)
		xpPrev := 0
		if lvl > 1 {
			xpPrev = xpTLWithCfg(lvl-1, cfg)
		}
		delta := xpForLevel - xpPrev
		if lvl == 1 {
			delta = 0
		}
		xpPerLevel[i] = delta
		if i == 0 {
			xpCumulative[i] = delta
		} else {
			xpCumulative[i] = xpCumulative[i-1] + delta
		}
	}

	writeJSON(w, http.StatusOK, APIResponse[progressionPreviewData]{
		Success: true,
		Data: progressionPreviewData{
			Levels:       levels,
			StatGains:    statGains,
			StatGainsAdj: statGainsAdj,
			HP:           hp,
			HPRaw:        hpRaw,
			Mana:         mana,
			ManaRaw:      manaRaw,
			XPPerLevel:   xpPerLevel,
			XPCumulative: xpCumulative,
		},
	})
}

// gainsForLevelWithCfg mirrors stats.StatInfo.GainsForLevel using a local cfg
// snapshot so the preview does not touch global config.
func gainsForLevelWithCfg(level, base int, cfg configs.ProgressionConfig) int {
	if level < 1 {
		level = 1
	}
	basePoints := int(math.Pow(float64(level-1), float64(cfg.BaseModExponent)) *
		float64(cfg.BaseModFactor) * float64(base))
	freePoints := int(math.Pow(float64(level), float64(cfg.NaturalGainsExponent)) *
		float64(cfg.NaturalGainsModFactor))
	return basePoints + freePoints
}

// applyCapWithCfg mirrors stats.StatInfo.Recalculate's compression step using a
// local cfg snapshot so the preview does not touch global config.
// value is treated as racial-only (no training/mods) matching the chart series.
func applyCapWithCfg(value int, cfg configs.ProgressionConfig) int {
	if value < int(cfg.StatCapThreshold) {
		return value
	}
	overage := value - int(cfg.StatCapAnchor)
	if overage < 0 {
		overage = 0
	}
	return int(cfg.StatCapAnchor) + int(math.Round(math.Pow(float64(overage), float64(cfg.StatCapExponent))*float64(cfg.StatCapScale)))
}

// xpTLWithCfg mirrors Character.XPTL using a local cfg snapshot (TNLScale=1.0).
func xpTLWithCfg(lvl int, cfg configs.ProgressionConfig) int {
	if lvl < 1 {
		lvl = 1
	}
	base := float64(cfg.XPBase)
	xp := base + math.Pow(float64(lvl), float64(cfg.XPLevelPower))*float64(cfg.XPLevelFactor)*base
	return int(xp)
}
