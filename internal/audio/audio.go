package audio

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type AudioConfig struct {
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	FilePath    string `yaml:"filepath,omitempty" json:"filepath,omitempty"`
	Volume      int    `yaml:"volume,omitempty" json:"volume,omitempty"`
}

var (
	audioLookup = map[string]AudioConfig{}
)

func GetFile(identifier string) AudioConfig {
	if f, ok := audioLookup[identifier]; ok {
		return f
	}
	return AudioConfig{}
}

func GetAllAudio() map[string]AudioConfig {
	cp := make(map[string]AudioConfig, len(audioLookup))
	for k, v := range audioLookup {
		cp[k] = v
	}
	return cp
}

// GetMusicFiles returns a sorted list of music filenames found under
// PublicHtml/static/audio/music/. Files whose names begin with "_" are skipped.
func GetMusicFiles() []string {
	publicHtml := configs.GetFilePathsConfig().PublicHtml.String()
	dir := filepath.Join(publicHtml, "static", "audio", "music")

	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{}
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "_") {
			continue
		}
		files = append(files, "static/audio/music/"+name)
	}
	sort.Strings(files)
	return files
}

func SaveAudio(entries map[string]AudioConfig) error {
	if entries == nil {
		return fmt.Errorf("audio entries cannot be nil")
	}

	path := util.FilePath(configs.GetFilePathsConfig().DataFiles.String() + `/audio.yaml`)

	bytes, err := yaml.Marshal(entries)
	if err != nil {
		return fmt.Errorf("marshaling audio config: %w", err)
	}

	if err := os.WriteFile(path, bytes, 0644); err != nil {
		return fmt.Errorf("writing audio config file: %w", err)
	}

	clear(audioLookup)
	for k, v := range entries {
		audioLookup[k] = v
	}

	return nil
}

func LoadAudioConfig() {

	start := time.Now()

	path := util.FilePath(string(configs.GetFilePathsConfig().DataFiles) + `/audio.yaml`)

	bytes, err := os.ReadFile(path)
	if err != nil {
		panic(errors.Wrap(err, `filepath: `+path))
	}

	clear(audioLookup)

	err = yaml.Unmarshal(bytes, &audioLookup)
	if err != nil {
		panic(errors.Wrap(err, `filepath: `+path))
	}

	mudlog.Info("...LoadAudioConfig()", "loadedCount", len(audioLookup), "Time Taken", time.Since(start))
}
