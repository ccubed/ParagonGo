package conversations

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

// ConversationFile identifies a single conversation file by zone and mob ID.
type ConversationFile struct {
	Zone  string `json:"zone"`
	MobId int    `json:"mob_id"`
}

// ConversationFileContents pairs a file's identity with its conversation entries.
type ConversationFileContents struct {
	ConversationFile
	Conversations []ConversationData `json:"conversations"`
}

func convFolder() string {
	return string(configs.GetFilePathsConfig().DataFiles) + `/conversations`
}

func convFilePath(zone string, mobId int) string {
	zone = ZoneNameSanitize(zone)
	return util.FilePath(convFolder() + `/` + fmt.Sprintf("%s/%d.yaml", zone, mobId))
}

// ConvFilePath returns the absolute path for a conversation file identified by
// zone and mobId. The zone name is sanitized before constructing the path.
func ConvFilePath(zone string, mobId int) string {
	return convFilePath(zone, mobId)
}

// ListConversationFiles returns a sorted list of every conversation file on disk.
func ListConversationFiles() ([]ConversationFile, error) {
	root := convFolder()

	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []ConversationFile{}, nil
		}
		return nil, fmt.Errorf("reading conversations directory: %w", err)
	}

	var result []ConversationFile
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		zone := entry.Name()
		zoneDir := filepath.Join(root, zone)
		files, err := os.ReadDir(zoneDir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".yaml") {
				continue
			}
			base := strings.TrimSuffix(f.Name(), ".yaml")
			mobId, err := strconv.Atoi(base)
			if err != nil {
				continue
			}
			result = append(result, ConversationFile{Zone: zone, MobId: mobId})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Zone != result[j].Zone {
			return result[i].Zone < result[j].Zone
		}
		return result[i].MobId < result[j].MobId
	})

	return result, nil
}

// GetConversationFile reads and returns the contents of a single conversation file.
func GetConversationFile(zone string, mobId int) (ConversationFileContents, error) {
	zone = ZoneNameSanitize(zone)
	path := convFilePath(zone, mobId)

	data, err := util.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ConversationFileContents{}, fmt.Errorf("conversation file not found: %s/%d", zone, mobId)
		}
		return ConversationFileContents{}, fmt.Errorf("reading conversation file: %w", err)
	}

	var convs []ConversationData
	if err := yaml.Unmarshal(data, &convs); err != nil {
		return ConversationFileContents{}, fmt.Errorf("parsing conversation file: %w", err)
	}

	return ConversationFileContents{
		ConversationFile: ConversationFile{Zone: zone, MobId: mobId},
		Conversations:    convs,
	}, nil
}

// SaveConversationFile validates and writes a conversation file to disk, then
// invalidates the in-memory existence cache for that entry.
func SaveConversationFile(zone string, mobId int, convs []ConversationData) error {
	zone = ZoneNameSanitize(zone)
	if zone == "" {
		return fmt.Errorf("zone is required")
	}
	if mobId <= 0 {
		return fmt.Errorf("mob_id must be a positive integer")
	}
	if len(convs) == 0 {
		return fmt.Errorf("conversations must not be empty")
	}
	for i, c := range convs {
		if len(c.Supported) == 0 {
			return fmt.Errorf("conversations[%d].Supported must not be empty", i)
		}
		if len(c.Conversation) == 0 {
			return fmt.Errorf("conversations[%d].Conversation must not be empty", i)
		}
	}

	dir := filepath.Join(convFolder(), zone)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating zone directory: %w", err)
	}

	path := convFilePath(zone, mobId)
	out, err := yaml.Marshal(convs)
	if err != nil {
		return fmt.Errorf("marshalling conversation data: %w", err)
	}

	if err := util.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("writing conversation file: %w", err)
	}

	invalidateCache(zone, mobId)
	return nil
}

// DeleteConversationFile removes a conversation file from disk and invalidates
// the in-memory existence cache for that entry.
func DeleteConversationFile(zone string, mobId int) error {
	zone = ZoneNameSanitize(zone)
	path := convFilePath(zone, mobId)

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("conversation file not found: %s/%d", zone, mobId)
		}
		return fmt.Errorf("removing conversation file: %w", err)
	}

	invalidateCache(zone, mobId)

	// Remove the zone directory if it is now empty.
	zoneDir := filepath.Join(convFolder(), zone)
	entries, err := os.ReadDir(zoneDir)
	if err == nil && len(entries) == 0 {
		_ = os.Remove(zoneDir)
	}

	return nil
}

// invalidateCache removes the cached existence result for a given zone/mob pair.
func invalidateCache(zone string, mobId int) {
	cacheKey := strconv.Itoa(mobId) + `-` + zone
	delete(converseCheckCache, cacheKey)
}
