package items

import (
	"errors"
	"strconv"
	"strings"
)

// ErrInvalidTarget means an Item target couldn't be read, e.g. "0.sword" or "2.".
var ErrInvalidTarget = errors.New("invalid item target")

// Target is a parsed Item target: what a player typed to pick out an Item.
// "sword" is the first match; "2.sword" is the second.
type Target struct {
	Keyword string // lower case; matches the start of any word in a display name
	Ordinal int    // which match to take, counting from 1
}

// ParseTarget reads "sword" or "N.sword". A prefix that isn't a number is part
// of the keyword, so "st.ives" is the keyword "st.ives".
func ParseTarget(input string) (Target, error) {
	input = strings.ToLower(strings.TrimSpace(input))
	target := Target{Keyword: input, Ordinal: 1}
	if dot := strings.Index(input, "."); dot > 0 {
		if n, err := strconv.Atoi(input[:dot]); err == nil {
			if n < 1 {
				return Target{}, ErrInvalidTarget
			}
			target = Target{Keyword: input[dot+1:], Ordinal: n}
		}
	}
	if target.Keyword == "" || strings.ContainsAny(target.Keyword, " \t") {
		return Target{}, ErrInvalidTarget
	}
	return target, nil
}

// Matches reports whether the keyword is the start of any word in name,
// ignoring case: "sw", "sword" and "training" all match "Training Sword".
func (t Target) Matches(name string) bool {
	for _, word := range strings.Fields(strings.ToLower(name)) {
		if strings.HasPrefix(word, t.Keyword) {
			return true
		}
	}
	return false
}

// Pick returns the Ordinal-th Item in list whose display name matches, counting
// in list order. Callers pass Items in arrival order so the numbers players use
// agree with what look and inventory show.
func (t Target) Pick(list []*ItemInstance, registry *ItemRegistry) (*ItemInstance, bool) {
	seen := 0
	for _, item := range list {
		if t.Matches(registry.DisplayName(item)) {
			seen++
			if seen == t.Ordinal {
				return item, true
			}
		}
	}
	return nil, false
}

// String is the target as a player would type it.
func (t Target) String() string {
	if t.Ordinal == 1 {
		return t.Keyword
	}
	return strconv.Itoa(t.Ordinal) + "." + t.Keyword
}
