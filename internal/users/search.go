package users

import (
	"bytes"
	"strconv"
	"strings"
)

// UserSearchResult holds the subset of user data returned by search endpoints.
type UserSearchResult struct {
	UserId   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Email    string `json:"email"`
}

// SearchUsers searches for users matching searchName.
//
// If searchName is a valid integer, the search matches by userId (exact match
// only). Otherwise it searches by username: an exact match returns a single
// result; all prefix matches are returned when there is no exact match. The
// username search is case-insensitive.
func SearchUsers(searchName string) []UserSearchResult {
	if searchName == "" {
		return nil
	}

	// Numeric input: match by userId.
	if uid, err := strconv.Atoi(searchName); err == nil && uid > 0 {
		return searchByUserId(uid)
	}

	return searchByUsername(searchName)
}

func searchByUserId(uid int) []UserSearchResult {
	type candidate struct {
		userId   int
		username string
	}

	var found *candidate

	GetUserIndex().ForEachRecord(func(rec IndexUserRecord) bool {
		if int(rec.UserID) == uid {
			c := candidate{userId: int(rec.UserID), username: string(bytes.TrimRight(rec.Username[:], "\x00"))}
			found = &c
			return false
		}
		return true
	})

	if found == nil {
		return nil
	}

	result := UserSearchResult{
		UserId:   found.userId,
		Username: found.username,
	}
	if u := GetByUserId(found.userId); u != nil {
		result.Role = u.Role
		result.Email = u.EmailAddress
	} else if loaded, err := LoadUser(found.username, true); err == nil {
		result.Role = loaded.Role
		result.Email = loaded.EmailAddress
	}
	return []UserSearchResult{result}
}

// SearchUsersByRole returns all users whose Role matches the given role string
// (case-insensitive). Results are capped at 500.
func SearchUsersByRole(role string) []UserSearchResult {
	if role == "" {
		return nil
	}
	needle := strings.ToLower(role)

	type candidate struct {
		userId   int
		username string
	}

	var matches []candidate
	GetUserIndex().ForEachRecord(func(rec IndexUserRecord) bool {
		if len(matches) >= 500 {
			return false
		}
		matches = append(matches, candidate{
			userId:   int(rec.UserID),
			username: string(bytes.TrimRight(rec.Username[:], "\x00")),
		})
		return true
	})

	results := make([]UserSearchResult, 0, len(matches))
	for _, c := range matches {
		var userRole, userEmail string
		if u := GetByUserId(c.userId); u != nil {
			userRole = u.Role
			userEmail = u.EmailAddress
		} else if loaded, err := LoadUser(c.username, true); err == nil {
			userRole = loaded.Role
			userEmail = loaded.EmailAddress
		}
		if strings.ToLower(userRole) != needle {
			continue
		}
		results = append(results, UserSearchResult{
			UserId:   c.userId,
			Username: c.username,
			Role:     userRole,
			Email:    userEmail,
		})
	}
	return results
}

func searchByUsername(searchName string) []UserSearchResult {
	needle := strings.ToLower(searchName)

	type candidate struct {
		userId   int
		username string
	}

	var exact *candidate
	var close []candidate

	GetUserIndex().ForEachRecord(func(rec IndexUserRecord) bool {
		raw := string(bytes.TrimRight(rec.Username[:], "\x00"))
		lower := strings.ToLower(raw)

		if lower == needle {
			c := candidate{userId: int(rec.UserID), username: raw}
			exact = &c
			return false
		}
		if strings.HasPrefix(lower, needle) && len(close) < 100 {
			close = append(close, candidate{userId: int(rec.UserID), username: raw})
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

	results := make([]UserSearchResult, 0, len(matches))
	for _, c := range matches {
		result := UserSearchResult{
			UserId:   c.userId,
			Username: c.username,
		}
		if u := GetByUserId(c.userId); u != nil {
			result.Role = u.Role
			result.Email = u.EmailAddress
		} else if loaded, err := LoadUser(c.username, true); err == nil {
			result.Role = loaded.Role
			result.Email = loaded.EmailAddress
		}
		results = append(results, result)
	}
	return results
}
