package buffs

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const (
	// Arbitrarily chosen round for calculating trigger round counts
	validationRound = 1000000
)

var (
	buffs map[int]*BuffSpec = make(map[int]*BuffSpec)

	validationCalculator gametime.GameDate
)

type BuffSpec struct {
	BuffId        int               // Unique identifier for this buff spec
	Name          string            // The name of the buff
	Description   string            // A description of the buff
	Secret        bool              // Whether or not the buff is secret (not displayed to the user)
	TriggerNow    bool              `yaml:"triggernow,omitempty"`    // if true, buff triggers once right when it is applied
	TriggerRate   string            `yaml:"triggerrate,omitempty"`   // How often should it trigger? (time string)
	RoundInterval int               `yaml:"roundinterval,omitempty"` // triggers every x rounds
	TriggerCount  int               `yaml:"triggercount,omitempty"`  // How many times it triggers before it is removed
	StatMods      statmods.StatMods `yaml:"statmods,omitempty"`      // stat mods for the duration of the buff
	Flags         []Flag            `yaml:"flags,omitempty"`         // A list of actions and such that this buff prevents or enables
}

// Calculates the value of this buff
func (b *BuffSpec) GetValue() int {
	val := 0

	for _, v := range b.StatMods {
		val += int(math.Abs(float64(v)))
	}

	freqVal := max(5-b.RoundInterval, 0)
	val += freqVal
	val += len(b.Flags) * 5

	if b.TriggerCount > 0 {
		val *= b.TriggerCount
	}

	return val
}

func (b *BuffSpec) VisibleNameDesc() (name, description string) {
	if b.Secret {
		return "Mysterious Affliction", "Unknown"
	}
	return b.Name, b.Description
}

type BuffMessage struct {
	User string
	Room string
}

type BuffMessages struct {
	Start  BuffMessage
	Effect BuffMessage
	End    BuffMessage
}

func GetBuffSpec(buffId int) *BuffSpec {
	if buffId < 0 {
		buffId *= -1
	}

	if buff, ok := buffs[buffId]; ok {
		return buff
	}

	return nil
}

func GetAllBuffIds() []int {

	var results []int = make([]int, 0, len(buffs))
	for _, buff := range buffs {
		results = append(results, buff.BuffId)
	}

	return results
}

// Searches for buffs whose name contains text and returns their Ids
func SearchBuffs(searchTerm string) []int {

	searchTerm = strings.TrimSpace(strings.ToLower(searchTerm))

	var results []int = make([]int, 0, 2)

	for _, buff := range buffs {
		if strings.Contains(strings.ToLower(buff.Name), searchTerm) {
			results = append(results, buff.BuffId)
		} else if strings.Contains(strings.ToLower(buff.Description), searchTerm) {
			results = append(results, buff.BuffId)
		}
	}

	return results
}

// Presumably to ensure the datafile hasn't messed something up.
func (b *BuffSpec) Id() int {
	return b.BuffId
}

// Presumably to ensure the datafile hasn't messed something up.
func (b *BuffSpec) Validate() error {

	// If this is the quit/meditating buff, override the trigger count
	if b.BuffId == 0 {
		b.TriggerCount = int(configs.GetNetworkConfig().LogoutRounds)
	}

	if validationCalculator.RoundNumber == 0 {
		validationCalculator = gametime.GetDate(validationRound)
	}

	b.RoundInterval = int(validationCalculator.AddPeriod(b.TriggerRate) - validationRound)

	if b.TriggerCount < 1 {
		return fmt.Errorf("buffId %d (%s) has a TriggersCount of < 1, must be at least 1", b.BuffId, b.Name)
	}
	if b.RoundInterval < 1 {
		return fmt.Errorf("buffId %d (%s) has a RoundInterval of < 1, must be at least 1. Is %s a valid time string?", b.BuffId, b.Name, b.TriggerRate)
	}
	return nil
}

func (b *BuffSpec) Filename() string {
	filename := util.ConvertForFilename(b.Name)
	return fmt.Sprintf("%d-%s.yaml", b.BuffId, filename)
}

func (b *BuffSpec) Filepath() string {
	return b.Filename()
}

func (b *BuffSpec) GetScript() string {

	// Check plugin-registered scripts first.
	if script := getPluginScript(b.BuffId); script != `` {
		return script
	}

	scriptPath := b.GetScriptPath()
	// Load the script into a string
	if _, err := os.Stat(scriptPath); err == nil {
		if bytes, err := util.ReadFile(scriptPath); err == nil {
			return string(bytes)
		}
	}

	return ``
}

func (b *BuffSpec) GetScriptPath() string {
	// Load any script for the buff (prefers .js, falls back to .lua)
	return util.ResolveScriptPath(string(configs.GetFilePathsConfig().DataFiles) + `/buffs/` + b.Filepath())
}

// file self loads due to init()
func LoadDataFiles() {

	start := time.Now()

	tmpBuffs, err := fileloader.LoadAllFlatFiles[int, *BuffSpec](string(configs.GetFilePathsConfig().DataFiles) + `/buffs`)
	if err != nil {
		panic(err)
	}

	buffs = tmpBuffs

	// Merge buffs from plugin file systems. Must happen here (not deferred)
	// so plugin buffs are present before items load and compute their values.
	loadPluginBuffs(buffs)

	mudlog.Info("buffSpec.LoadDataFiles()", "loadedCount", len(buffs), "Time Taken", time.Since(start))
}
