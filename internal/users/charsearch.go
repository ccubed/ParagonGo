package users

import (
	"strings"
)

// CharacterSearchResult holds the data returned by character search endpoints.
type CharacterSearchResult struct {
	UserId        int    `json:"user_id"`
	Username      string `json:"username"`
	CharacterName string `json:"character_name"`
}

// SearchCharacters searches the character index for names matching searchName.
// An exact match (case-insensitive) returns a single result. When there is no
// exact match, all names that have searchName as a prefix are returned (up to
// 100). The results include the owning userId and username.
func SearchCharacters(searchName string) []CharacterSearchResult {
	if searchName == "" {
		return nil
	}

	needle := strings.ToLower(searchName)

	type candidate struct {
		name   string
		userId int
	}

	var exact *candidate
	var close []candidate

	GetCharacterIndex().ForEach(func(name string, userId int) bool {
		if name == needle {
			c := candidate{name: name, userId: userId}
			exact = &c
			return false
		}
		if strings.HasPrefix(name, needle) && len(close) < 100 {
			close = append(close, candidate{name: name, userId: userId})
		}
		return true
	})

	var matches []candidate
	if exact != nil {
		matches = []candidate{*exact}
	} else {
		matches = close
	}

	if len(matches) == 0 {
		return nil
	}

	results := make([]CharacterSearchResult, 0, len(matches))
	for _, c := range matches {
		username, _ := GetUserIndex().FindByUserId(c.userId)
		results = append(results, CharacterSearchResult{
			UserId:        c.userId,
			Username:      username,
			CharacterName: c.name,
		})
	}
	return results
}
