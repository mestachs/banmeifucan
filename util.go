package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"unicode"
)

// Cached regex patterns
var (
	uuidPattern       = "[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}" // UUID pattern
	longIntPattern    = `\d+`                                                                 // Match long integers
	dhis2UidPattern   = `[a-zA-Z]{1}[a-zA-Z0-9]{10}`                                          // DHIS2 UID pattern (alphanumeric)
	enketoUidPattern  = `[a-zA-Z0-9]{8}`                                                      // enketo pattern (alphanumeric)
	compiledRegex     *regexp.Regexp
	compiledRegexOnce sync.Once
)

// getCompiledRegex ensures the regex is compiled only once
func getCompiledRegex() *regexp.Regexp {
	compiledRegexOnce.Do(func() {
		// Combine the patterns into a single regular expression
		combinedPattern := fmt.Sprintf("^(%s|%s|%s|%s)$", uuidPattern, longIntPattern, enketoUidPattern, dhis2UidPattern)
		var err error
		log.Printf("{id} identification pattern %s", combinedPattern)
		compiledRegex, err = regexp.Compile(combinedPattern)
		if err != nil {
			panic("failed to compile regex: " + err.Error())
		}
	})
	return compiledRegex
}

// CommonPrefixes contains some common prefixes found in human-readable strings
var CommonPrefixes = []string{"data", "user", "file", "element", "info", "image", "content", "user", "name", "proj", "chil"}

// CommonSuffixes contains some common suffixes found in human-readable strings
var CommonSuffixes = []string{"ing", "ed", "able", "ment", "tion", "ness", "ize", "ers"}

// IsCamelCase checks if a string follows CamelCase or PascalCase
func IsCamelCase(s string) bool {
	// Match camelCase or PascalCase: e.g., "dataElements", "userName"
	re := regexp.MustCompile(`[a-z]+([A-Z][a-z]+)+`)
	return re.MatchString(s)
}

// CheckCommonPrefixes checks if the string contains common prefixes
func CheckCommonPrefixes(s string) bool {
	s = strings.ToLower(s)
	for _, prefix := range CommonPrefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

// CheckCommonSuffixes checks if the string contains common suffixes
func CheckCommonSuffixes(s string) bool {
	s = strings.ToLower(s)
	for _, suffix := range CommonSuffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

func countUpperAndLower(s string) (int, int) {
	upperCount := 0
	lowerCount := 0

	for _, char := range s {
		if unicode.IsUpper(char) {
			upperCount++
		} else if unicode.IsLower(char) {
			lowerCount++
		}
	}

	return upperCount, lowerCount
}

func containsLettersAndNumbers(s string) bool {
	hasLetter := false
	hasDigit := false

	// Iterate over each character in the string
	for _, char := range s {
		// Check if the character is a letter
		if unicode.IsLetter(char) {
			hasLetter = true
		}
		// Check if the character is a digit
		if unicode.IsDigit(char) {
			hasDigit = true
		}
		// If both conditions are true, return true
		if hasLetter && hasDigit {
			return true
		}
	}

	// Return false if we haven't found both a letter and a digit
	return false
}

// CheckStringPattern checks if the string follows common linguistic patterns
func CheckStringPattern(s string) bool {

	if containsLettersAndNumbers(s) {
		return false
	}

	upperCount, _ := countUpperAndLower(s)
	upperLowerRatio := float64(upperCount) / float64(len(s))
	if upperLowerRatio >= 0.35 {
		return false
	}

	// Check if string is CamelCase or PascalCase
	if IsCamelCase(s) {
		return true
	}

	// Check for common prefixes or suffixes
	if CheckCommonPrefixes(s) || CheckCommonSuffixes(s) {
		return true
	}

	// The string doesn't seem to follow common patterns, likely random
	return false
}

// cleanPath replaces dynamic path segments (UUIDs, long integers, DHIS2 UIDs) with {id}
func CleanPath(path string) string {
	// Get the cached compiled regex
	re := getCompiledRegex()

	segments := strings.Split(path, "/")

	for i, segment := range segments {
		if re.MatchString(segment) {
			if len(segment) == 11 || len(segment) == 8 {
				check := CheckStringPattern(segment)
				if !check {
					segments[i] = "{id}"
				}
			} else {
				segments[i] = "{id}"
			}
		}
	}
	// Join the segments back into a path
	return strings.Join(segments, "/") // Join without leading "/"

}
