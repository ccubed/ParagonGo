package quests

import (
	"fmt"
	"os"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// SaveQuest persists a Quest to its data file and updates the in-memory cache.
// The QuestId must already be set.
func SaveQuest(q *Quest) error {
	if q.QuestId < 1 {
		return fmt.Errorf("cannot save quest with invalid QuestId %d", q.QuestId)
	}

	if err := q.Validate(); err != nil {
		return err
	}

	saveModes := []fileloader.SaveOption{}
	if configs.GetFilePathsConfig().CarefulSaveFiles {
		saveModes = append(saveModes, fileloader.SaveCareful)
	}

	if err := fileloader.SaveFlatFile[*Quest](configs.GetFilePathsConfig().DataFiles.String()+`/quests`, q, saveModes...); err != nil {
		return err
	}

	quests[q.QuestId] = q
	return nil
}

// DeleteQuest removes a quest from disk and from the in-memory cache.
func DeleteQuest(questId int) error {
	q, ok := quests[questId]
	if !ok {
		return fmt.Errorf("quest %d not found", questId)
	}

	yamlPath := util.FilePath(configs.GetFilePathsConfig().DataFiles.String() + `/quests/` + q.Filepath())
	if err := os.Remove(yamlPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing quest yaml: %w", err)
	}

	delete(quests, questId)
	return nil
}

// GetQuestById returns the quest with the given id, or nil if not found.
func GetQuestById(questId int) *Quest {
	return quests[questId]
}
