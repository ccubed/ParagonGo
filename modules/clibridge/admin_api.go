package clibridge

import (
	"encoding/json"
	"net/http"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

const listConfigKey = "list-config"

// listConfig holds the slice-typed config fields that cannot go through SetVal.
type listConfig struct {
	AllowedTools []string `yaml:"AllowedTools"`
	AllowedPaths []string `yaml:"AllowedPaths"`
}

func (m *CLIBridgeModule) loadListConfig() listConfig {
	var lc listConfig
	m.plug.ReadIntoStruct(listConfigKey, &lc)
	return lc
}

func (m *CLIBridgeModule) saveListConfig(lc listConfig) error {
	return m.plug.WriteStruct(listConfigKey, lc)
}

// apiGetConfig handles GET /admin/api/v1/clibridge-config.
func (m *CLIBridgeModule) apiGetConfig(r *http.Request) (int, bool, any) {
	lc := m.loadListConfig()

	// Seed from overlay defaults if plugin storage is empty.
	if len(lc.AllowedTools) == 0 {
		lc.AllowedTools = m.getAllowedTools()
	}
	if len(lc.AllowedPaths) == 0 {
		lc.AllowedPaths = m.getAllowedPaths()
	}

	enabled := m.isEnabled()

	return http.StatusOK, true, map[string]any{
		"Enabled":      enabled,
		"AllowedTools": lc.AllowedTools,
		"AllowedPaths": lc.AllowedPaths,
	}
}

// apiPatchConfig handles PATCH /admin/api/v1/clibridge-config.
func (m *CLIBridgeModule) apiPatchConfig(r *http.Request) (int, bool, any) {
	var body struct {
		Enabled      *bool    `json:"Enabled"`
		AllowedTools []string `json:"AllowedTools"`
		AllowedPaths []string `json:"AllowedPaths"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return http.StatusBadRequest, false, "malformed request body: " + err.Error()
	}

	if body.Enabled != nil {
		val := "false"
		if *body.Enabled {
			val = "true"
		}
		_ = configs.SetVal("Modules.clibridge.Enabled", val)
	}

	lc := listConfig{
		AllowedTools: body.AllowedTools,
		AllowedPaths: body.AllowedPaths,
	}
	if err := m.saveListConfig(lc); err != nil {
		return http.StatusInternalServerError, false, "failed to save config: " + err.Error()
	}

	return http.StatusOK, true, "saved"
}
