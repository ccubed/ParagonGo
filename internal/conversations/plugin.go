package conversations

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/fileloader"
)

var (
	pluginFileSystems []fileloader.ReadableGroupFS
)

// RegisterFS registers a plugin file system to be searched when looking up
// conversation data files. Must be called before conversations are requested.
func RegisterFS(f ...fileloader.ReadableGroupFS) {
	pluginFileSystems = append(pluginFileSystems, f...)
}

// pluginConversationKey builds the relative path used to look up a conversation
// file inside a plugin file system. It matches the on-disk layout
// (conversations/<zone>/<mobId>.yaml) minus the data-files root, which is the
// same short-path form AttachFileSystem registers.
func pluginConversationKey(zone string, mobId int) string {
	return fmt.Sprintf(`conversations/%s/%d.yaml`, zone, mobId)
}

// readPluginConversationFile returns the bytes of a conversation file provided
// by a registered plugin file system, or (nil, false) if none provide it.
// zone is expected to already be sanitized via ZoneNameSanitize.
func readPluginConversationFile(zone string, mobId int) ([]byte, bool) {
	key := pluginConversationKey(zone, mobId)
	for _, groupFS := range pluginFileSystems {
		if b, err := groupFS.ReadFile(key); err == nil {
			return b, true
		}
	}
	return nil, false
}

// hasPluginConversationFile reports whether any registered plugin file system
// provides a conversation file for the given sanitized zone and mob id.
func hasPluginConversationFile(zone string, mobId int) bool {
	_, ok := readPluginConversationFile(zone, mobId)
	return ok
}
