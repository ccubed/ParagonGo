package telemetry

import "github.com/GoMudEngine/GoMud/internal/util"

func GetMemoryUsage() map[string]util.MemoryResult {
	ret := map[string]util.MemoryResult{}
	ret["records"] = util.MemoryResult{Memory: util.MemoryUsage(records), Count: len(records)}
	ret["index"] = util.MemoryResult{Memory: util.MemoryUsage(index), Count: len(index)}
	return ret
}

func init() {
	util.AddMemoryReporter(`Telemetry`, GetMemoryUsage)
}
