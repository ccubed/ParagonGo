package characters

import (
	"fmt"
	"strings"
)

type KDStats struct {
	TotalKills  int         `json:"totalkills,omitempty"`  // Quick tally of kills
	Kills       map[int]int `json:"kills,omitempty"`       // map of MobId to count
	TotalDeaths int         `json:"totaldeaths,omitempty"` // Quick tally of deaths

	TotalPvpKills  int            `json:"totalpvpkills,omitempty"`  // Quick tally of pvp kills
	PlayerKills    map[string]int `json:"playerkills,omitempty"`    // map of userid:username to count
	PlayerDeaths   map[string]int `json:"playerdeaths,omitempty"`   // map of userid:username to count
	TotalPvpDeaths int            `json:"totalpvpdeaths,omitempty"` // Quick tally of pvp deaths

	EliteKills  map[string]int `json:"elitekills,omitempty"`  // map of mobId:mobName to count (not included in TotalKills)
	EliteDeaths map[string]int `json:"elitedeaths,omitempty"` // map of mobId:mobName to count of times killed by an elite
}

func (kd *KDStats) GetMobKDRatio() float64 {
	if kd.TotalDeaths == 0 {
		return float64(kd.TotalKills)
	}
	return float64(kd.TotalKills) / float64(kd.TotalDeaths)
}

func (kd *KDStats) GetPvpKDRatio() float64 {
	if kd.TotalPvpDeaths == 0 {
		return float64(kd.TotalPvpKills)
	}
	return float64(kd.TotalPvpKills) / float64(kd.TotalPvpDeaths)
}

func (kd *KDStats) GetMobKills(mobId ...int) int {
	if len(mobId) == 0 {
		return kd.TotalKills
	}

	if kd.Kills == nil {
		kd.Kills = make(map[int]int)
	}

	total := 0
	for _, id := range mobId {
		total += kd.Kills[id]
	}
	return total
}

func (kd *KDStats) AddPlayerKill(killedUserId int, killedCharName string) {
	if kd.PlayerKills == nil {
		kd.PlayerKills = make(map[string]int)
	}

	keyName := fmt.Sprintf(`%d:%s`, killedUserId, killedCharName)

	kd.TotalPvpKills++
	kd.PlayerKills[keyName] = kd.PlayerKills[keyName] + 1
}

func (kd *KDStats) AddPlayerDeath(killedByUserId int, killedByCharName string) {
	if kd.PlayerDeaths == nil {
		kd.PlayerDeaths = make(map[string]int)
	}

	keyName := fmt.Sprintf(`%d:%s`, killedByUserId, killedByCharName)
	kd.PlayerDeaths[keyName] = kd.PlayerDeaths[keyName] + 1
}

func (kd *KDStats) AddMobKill(mobId int) {
	if kd.Kills == nil {
		kd.Kills = make(map[int]int)
	}
	kd.TotalKills++
	kd.Kills[mobId] = kd.Kills[mobId] + 1
}

func (kd *KDStats) GetMobDeaths() int {
	return kd.TotalDeaths
}

func (kd *KDStats) GetPvpDeaths() int {
	return kd.TotalPvpDeaths
}

func (kd *KDStats) AddMobDeath() {
	kd.TotalDeaths++
}

func (kd *KDStats) AddPvpDeath() {
	kd.TotalPvpDeaths++
}

func (kd *KDStats) AddEliteKill(mobId int, mobName string) {
	if kd.EliteKills == nil {
		kd.EliteKills = make(map[string]int)
	}
	key := fmt.Sprintf(`%d:%s`, mobId, mobName)
	kd.EliteKills[key] = kd.EliteKills[key] + 1
}

func (kd *KDStats) AddEliteDeath(mobId int, mobName string) {
	if kd.EliteDeaths == nil {
		kd.EliteDeaths = make(map[string]int)
	}
	key := fmt.Sprintf(`%d:%s`, mobId, mobName)
	kd.EliteDeaths[key] = kd.EliteDeaths[key] + 1
}

func (kd *KDStats) GetEliteKills(mobId ...int) int {
	if len(mobId) == 0 {
		total := 0
		for _, v := range kd.EliteKills {
			total += v
		}
		return total
	}
	total := 0
	for key, v := range kd.EliteKills {
		for _, id := range mobId {
			if strings.HasPrefix(key, fmt.Sprintf(`%d:`, id)) {
				total += v
			}
		}
	}
	return total
}
