package uasanitizer

import (
	"github.com/hashicorp/go-secure-stdlib/strutil"
)

const (
	validCharacters = "[A-Z]|[a-z]|[0-9]|/|\\.| |;|:|\\+|\\(|\\)|;|_|,"
)

// Replace replaces all non-valid character from the provided userAgent with replaceWith and returns the the updates
// userAgent and if any changes have been done.
func Replace(userAgent string, replaceWith byte) (string, bool) {
	userAgent, changed := strutil.ReplaceNonMatcher(userAgent, validCharacters, replaceWith)
	return userAgent, changed > 0
}

// Remove removes all non-valid character from the provided userAgent and returns the the updates
// userAgent and if any changes have been done.
func Remove(userAgent string) (string, bool) {
	userAgent, changed := strutil.RemoveNonMatcher(userAgent, validCharacters)
	return userAgent, changed > 0
}
