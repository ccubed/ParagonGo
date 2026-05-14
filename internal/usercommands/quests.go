package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Quests(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	showHidden := rest == `all+`
	showComplete := (rest == `all`) || showHidden

	allQuestProgress := user.Character.GetQuestProgress()

	if rest == `all+` {
		for _, quest := range quests.GetAllQuests() {
			if _, ok := allQuestProgress[quest.QuestId]; ok {
				continue
			}
			allQuestProgress[quest.QuestId] = `all+`
		}
	}

	var rows []questRow
	questsTotal := 0

	for questId, questStep := range allQuestProgress {
		questToken := quests.PartsToToken(questId, questStep)
		if questInfo := quests.GetQuest(questToken); questInfo != nil {

			if !showHidden && questInfo.Secret {
				continue
			}

			questsTotal++

			totalSteps := len(questInfo.Steps)
			completedSteps := 0
			description := questInfo.Description

			if questStep != `all+` {
				for _, step := range questInfo.Steps {
					completedSteps++
					if step.Id == questStep {
						description = step.Description
						break
					}
				}
			}

			completion := float64(completedSteps) / float64(totalSteps)

			if !showComplete && completion >= 1 {
				continue
			}

			rows = append(rows, questRow{
				id:          questInfo.QuestId,
				name:        questInfo.Name,
				description: description,
				completion:  completion,
			})
		}
	}

	// Sort by quest id
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && rows[j].id < rows[j-1].id; j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}

	user.SendText(buildQuestsPanel(rows, len(rows), questsTotal))

	return true, nil
}
