package uasanitizer

import (
	"regexp"
	"testing"
)

func TestReplace(t *testing.T) {
	userAgent := "Mozilla/5.0 '\"<h1>"
	userAgentWant := regexp.MustCompile(`Mozilla/5.0 ___h1_`)
	sanitized, success := Replace(userAgent, '_')
	if !userAgentWant.MatchString(sanitized) || !success {
		t.Fatalf(`Replace("Mozilla/5.0 '\"<h1>",'_') = %q, %v want match for %#q, true`, sanitized, success, userAgentWant)
	}
}

func TestRemove(t *testing.T) {
	userAgent := "Mozilla/5.0 '\"<h1>"
	userAgentWant := regexp.MustCompile(`Mozilla/5.0 h1`)
	sanitized, success := Remove(userAgent)
	if !userAgentWant.MatchString(sanitized) || !success {
		t.Fatalf(`Replace("Mozilla/5.0 '\"<h1>",'_') = %q, %v want match for %#q, true`, sanitized, success, userAgentWant)
	}
}
