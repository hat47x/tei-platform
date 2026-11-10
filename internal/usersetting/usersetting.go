// Package usersetting binds a declared setting Data to the values a caller
// reads, so a setting key is something the model names rather than a string a
// script invents.
//
// X-TEA's script API wrote arbitrary keys: setVariant set a system variable and
// setUserValueOf set a value for the current user (D:Session.java L2112,
// D:XFScriptable.java L56–60). Nothing listed which keys existed, so a misspelt
// key read as "unset" and a key one screen wrote was invisible to the next. Here
// the keys are rows of a declared Data, and a key that is not declared is
// reported instead of read as empty.
package usersetting

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/ir"
)

// Attribute roles a setting Data must declare.
const (
	// RoleKey names the setting: the identifier a caller reads by.
	RoleKey = "setting-key"
	// RoleValue is the value the setting holds.
	RoleValue = "setting-value"
	// RoleUser is optional. When declared, a row belongs to one user, and a row
	// without a user is a shared default. That is the difference X-TEA drew
	// between a system variant and a user value.
	RoleUser = "setting-user"
)

// Declaration is the attributes a setting Data declares.
type Declaration struct {
	DataID core.CanonicalID
	Key    core.CanonicalID
	Value  core.CanonicalID
	User   core.CanonicalID
}

// Resolve reads the setting declaration from the world. The key and the value are
// required: without them a row cannot be read as a setting. The user is optional,
// because a table of shared settings does not have to name one.
func Resolve(world ir.World, dataID core.CanonicalID) (Declaration, error) {
	for _, namespace := range world.Semantics {
		for _, data := range namespace.Data {
			if data.ID != dataID {
				continue
			}
			declaration := Declaration{DataID: data.ID}
			for _, attribute := range data.Attributes {
				switch attribute.Role {
				case RoleKey:
					declaration.Key = attribute.ID
				case RoleValue:
					declaration.Value = attribute.ID
				case RoleUser:
					declaration.User = attribute.ID
				}
			}
			var missing []string
			if declaration.Key == "" {
				missing = append(missing, RoleKey)
			}
			if declaration.Value == "" {
				missing = append(missing, RoleValue)
			}
			if len(missing) > 0 {
				return Declaration{}, fmt.Errorf("setting data %s does not declare %v", dataID, missing)
			}
			return declaration, nil
		}
	}
	return Declaration{}, fmt.Errorf("setting data %s is not declared", dataID)
}

// Setting is one declared setting, read from a row.
type Setting struct {
	Key   string
	Value string
	// User is empty for a shared setting.
	User string
}

// Bind turns stored rows into settings. A row without a key is reported with its
// position, because a keyless row cannot be read by anyone and silently dropping
// it would hide a broken table.
func (d Declaration) Bind(rows []map[core.CanonicalID]string) ([]Setting, error) {
	out := make([]Setting, 0, len(rows))
	for index, row := range rows {
		key := row[d.Key]
		if key == "" {
			return nil, fmt.Errorf("setting row %d: %s is empty", index+1, d.Key)
		}
		setting := Setting{Key: key, Value: row[d.Value]}
		if d.User != "" {
			setting.User = row[d.User]
		}
		out = append(out, setting)
	}
	return out, nil
}

// Settings is the bound settings of one Data, indexed for lookup.
type Settings struct {
	// hasUser is true when the Data declares a user role. A Data without one is a
	// shared table, so a caller's user does not select a row.
	hasUser bool
	shared  map[string]string
	byUser  map[string]map[string]string
}

// NewSettings indexes bound settings.
//
// Two rows for the same key and the same user are refused: the value would depend
// on row order, and a caller would read one of them without knowing which.
func NewSettings(hasUser bool, settings []Setting) (Settings, error) {
	bound := Settings{hasUser: hasUser, shared: map[string]string{}, byUser: map[string]map[string]string{}}
	for _, setting := range settings {
		if !hasUser || setting.User == "" {
			if _, exists := bound.shared[setting.Key]; exists {
				return Settings{}, fmt.Errorf("setting %s is declared twice", setting.Key)
			}
			bound.shared[setting.Key] = setting.Value
			continue
		}
		userValues := bound.byUser[setting.User]
		if userValues == nil {
			userValues = map[string]string{}
			bound.byUser[setting.User] = userValues
		}
		if _, exists := userValues[setting.Key]; exists {
			return Settings{}, fmt.Errorf("setting %s is declared twice for user %s", setting.Key, setting.User)
		}
		userValues[setting.Key] = setting.Value
	}
	return bound, nil
}

// Value returns the setting's value for the user.
//
// A user's own row wins over the shared one, which is what setUserValueOf did for
// a logged-in user. A key that is declared for another user only is reported
// rather than read as empty: X-TEA could not tell a missing key from an unset
// value, so a typo read as the empty string and the screen quietly lost its
// setting.
func (s Settings) Value(key, user string) (string, error) {
	if values, ok := s.byUser[user]; ok {
		if value, ok := values[key]; ok {
			return value, nil
		}
	}
	if value, ok := s.shared[key]; ok {
		return value, nil
	}
	if len(s.byUser) > 0 {
		for _, values := range s.byUser {
			if _, ok := values[key]; ok {
				return "", fmt.Errorf("setting %s is not declared for user %q", key, user)
			}
		}
	}
	return "", fmt.Errorf("setting %s is not declared", key)
}

// Keys returns the declared keys, shared first and then the users' keys, so a
// caller can list what it may read without knowing them by heart.
func (s Settings) Keys() []string {
	seen := map[string]bool{}
	var out []string
	for key := range s.shared {
		if !seen[key] {
			out = append(out, key)
			seen[key] = true
		}
	}
	for _, values := range s.byUser {
		for key := range values {
			if !seen[key] {
				out = append(out, key)
				seen[key] = true
			}
		}
	}
	sortStrings(out)
	return out
}

// sortStrings keeps the key list stable so a caller can compare two runs.
func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
