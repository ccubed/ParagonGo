package configs

import (
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

// --- Helpers ---

func seedConfig() Config {
	c := Config{}
	c.Server.MudName = "BenchMUD"
	c.Server.Seed = "benchseed"
	c.Server.MaxCPUCores = 4
	c.Server.CurrentVersion = "1.0.0"
	c.Timing.TurnMs = 100
	c.Timing.RoundSeconds = 4
	c.Timing.RoundsPerAutoSave = 900
	c.Timing.RoundsPerDay = 20
	c.Timing.NightHours = 8
	c.FilePaths.DataFiles = "_datafiles/world/default"
	c.FilePaths.HttpsCacheDir = "_datafiles/tls"
	c.GamePlay.PVP.Enabled = PVPEnabled
	c.GamePlay.XPScale = 100
	c.GamePlay.ShopRestockRate = "6 hours"
	c.GamePlay.ContainerSizeMax = 10
	c.GamePlay.PricePerLife = 100
	c.GamePlay.Death.XPPenalty = "none"
	c.GamePlay.Death.CorpseDecayTime = "1 hour"
	c.Validation.NameSizeMin = 3
	c.Validation.NameSizeMax = 20
	c.Validation.PasswordSizeMin = 4
	c.Validation.PasswordSizeMax = 16
	c.Validation.EmailOnJoin = "optional"
	c.Validation.BannedNames = ConfigSliceString{"admin", "moderator", "god*", "system*", "test*"}
	c.Modules = Modules{"webclient": map[string]any{"enabled": true}}
	return c
}

func seedOverrideMap(n int) map[string]any {
	m := make(map[string]any)
	m["Server.MudName"] = ConfigString("OverrideMUD")
	m["Timing.TurnMs"] = ConfigInt(50)
	m["GamePlay.XPScale"] = ConfigFloat(150)
	m["Validation.NameSizeMax"] = ConfigInt(30)
	for i := 0; i < n; i++ {
		m[fmt.Sprintf("Modules.bench%d.key", i)] = fmt.Sprintf("val%d", i)
	}
	return m
}

// --- Validate ---

func Benchmark_Validate(b *testing.B) {
	c := seedConfig()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		c.validated = false
		c.Validate()
	}
}

// --- DotPaths (reflection walk) ---

func Benchmark_DotPaths(b *testing.B) {
	c := seedConfig()
	c.Validate()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = c.DotPaths()
	}
}

// --- AllConfigData (DotPaths + filter) ---

func Benchmark_AllConfigData_NoFilter(b *testing.B) {
	c := seedConfig()
	c.Validate()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = c.AllConfigData()
	}
}

func Benchmark_AllConfigData_WithExcludes(b *testing.B) {
	c := seedConfig()
	c.Validate()
	excludes := []string{"*secret*", "*password*", "*key*"}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = c.AllConfigData(excludes...)
	}
}

// --- OverlayOverrides (YAML round-trip) ---

func Benchmark_OverlayOverrides_Small(b *testing.B) {
	dotMap := map[string]any{
		"Server.MudName": "NewName",
		"Timing.TurnMs":  50,
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		c := seedConfig()
		_ = c.OverlayOverrides(dotMap)
	}
}

func Benchmark_OverlayOverrides_Medium(b *testing.B) {
	dotMap := seedOverrideMap(10)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		c := seedConfig()
		_ = c.OverlayOverrides(dotMap)
	}
}

func Benchmark_OverlayOverrides_Large(b *testing.B) {
	dotMap := seedOverrideMap(100)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		c := seedConfig()
		_ = c.OverlayOverrides(dotMap)
	}
}

// --- unflattenMap ---

func Benchmark_UnflattenMap_Small(b *testing.B) {
	flat := map[string]any{
		"Server.MudName": "Test",
		"Timing.TurnMs":  100,
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = unflattenMap(flat)
	}
}

func Benchmark_UnflattenMap_Large(b *testing.B) {
	flat := seedOverrideMap(100)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = unflattenMap(flat)
	}
}

// --- Flatten ---

func Benchmark_Flatten_Shallow(b *testing.B) {
	nested := map[string]any{
		"Server": map[string]any{
			"MudName":     "Test",
			"MaxCPUCores": 4,
		},
		"Timing": map[string]any{
			"TurnMs":       100,
			"RoundSeconds": 4,
		},
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = Flatten(nested)
	}
}

func Benchmark_Flatten_Deep(b *testing.B) {
	nested := map[string]any{
		"Server": map[string]any{
			"MudName": "Test",
		},
		"GamePlay": map[string]any{
			"Death": map[string]any{
				"XPPenalty":           "none",
				"EquipmentDropChance": 0.5,
				"ProtectionLevels":    3,
				"CorpsesEnabled":      true,
				"CorpseDecayTime":     "1 hour",
			},
			"Party": map[string]any{
				"MaxPlayerCount": 5,
				"SameRoomOnly":   true,
			},
		},
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = Flatten(nested)
	}
}

// --- Flatten + unflatten round-trip ---

func Benchmark_Flatten_Unflatten_RoundTrip(b *testing.B) {
	flat := seedOverrideMap(50)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		nested := unflattenMap(flat)
		_ = Flatten(nested)
	}
}

// --- YAML marshal/unmarshal (isolates the cost inside OverlayOverrides) ---

func Benchmark_YAMLMarshalUnmarshal(b *testing.B) {
	dotMap := seedOverrideMap(10)
	nested := unflattenMap(dotMap)
	c := seedConfig()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		bytes, _ := yaml.Marshal(nested)
		_ = yaml.Unmarshal(bytes, &c)
	}
}

// --- FindFullPath (key lookup) ---

func Benchmark_FindFullPath_Exact(b *testing.B) {
	c := seedConfig()
	c.Validate()

	keyLookups = map[string]string{}
	typeLookups = map[string]string{}
	for k, v := range c.AllConfigData() {
		keyLookups[k] = k
		typeLookups[k] = fmt.Sprintf("%T", v)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		FindFullPath("Server.MudName")
	}
}

func Benchmark_FindFullPath_Suffix(b *testing.B) {
	c := seedConfig()
	c.Validate()

	keyLookups = map[string]string{}
	typeLookups = map[string]string{}
	for k, v := range c.AllConfigData() {
		keyLookups[strings.ToLower(k)] = k
		typeLookups[k] = fmt.Sprintf("%T", v)

		parts := strings.Split(k, ".")
		for i := len(parts) - 1; i >= 0; i-- {
			suffix := strings.Join(parts[i:], ".")
			keyLookups[strings.ToLower(suffix)] = k
			noDotsuffix := strings.Join(parts[i:], "")
			keyLookups[strings.ToLower(noDotsuffix)] = k
		}
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		FindFullPath("mudname")
	}
}

func Benchmark_FindFullPath_Miss(b *testing.B) {
	c := seedConfig()
	c.Validate()

	keyLookups = map[string]string{}
	typeLookups = map[string]string{}
	for k, v := range c.AllConfigData() {
		keyLookups[strings.ToLower(k)] = k
		typeLookups[k] = fmt.Sprintf("%T", v)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		FindFullPath("nonexistent.key.path")
	}
}

// --- IsBannedName ---

func Benchmark_IsBannedName_NoMatch(b *testing.B) {
	c := seedConfig()
	c.Validate()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		c.IsBannedName("playerone")
	}
}

func Benchmark_IsBannedName_Match(b *testing.B) {
	c := seedConfig()
	c.Validate()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		c.IsBannedName("admin")
	}
}

func Benchmark_IsBannedName_WildcardMatch(b *testing.B) {
	c := seedConfig()
	c.Validate()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		c.IsBannedName("godlike")
	}
}

func Benchmark_IsBannedName_LargeList(b *testing.B) {
	c := seedConfig()
	names := make(ConfigSliceString, 200)
	for i := range names {
		names[i] = fmt.Sprintf("banned%d*", i)
	}
	c.Validation.BannedNames = names
	c.Validate()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		c.IsBannedName("playerone")
	}
}

// --- GetConfig (read-lock + copy) ---

func Benchmark_GetConfig(b *testing.B) {
	configDataLock.Lock()
	configData = seedConfig()
	configData.Validate()
	configDataLock.Unlock()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = GetConfig()
	}
}

// --- GetConfig under contention ---

func Benchmark_GetConfig_Parallel(b *testing.B) {
	configDataLock.Lock()
	configData = seedConfig()
	configData.Validate()
	configDataLock.Unlock()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = GetConfig()
		}
	})
}

// --- StringToConfigValue ---

func Benchmark_StringToConfigValue_KnownType(b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = StringToConfigValue("42", "configs.ConfigInt")
	}
}

func Benchmark_StringToConfigValue_UnknownType(b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = StringToConfigValue("42", "")
	}
}

func Benchmark_StringToConfigValue_Bool(b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = StringToConfigValue("true", "configs.ConfigBool")
	}
}

func Benchmark_StringToConfigValue_SliceString(b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = StringToConfigValue("one,two,three,four,five", "configs.ConfigSliceString")
	}
}

// --- SetOverrides (full overlay pipeline) ---

func Benchmark_SetOverrides(b *testing.B) {
	overrideMap := seedOverrideMap(10)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		c := seedConfig()
		_ = c.SetOverrides(overrideMap)
	}
}

// --- End-to-end: overlay + validate ---

func Benchmark_OverlayThenValidate(b *testing.B) {
	dotMap := seedOverrideMap(10)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		c := seedConfig()
		_ = c.OverlayOverrides(dotMap)
		c.Validate()
	}
}
