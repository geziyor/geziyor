package internal

// PreferFirst returns first non-empty string
func PreferFirst(first string, second string) string {
	if first != "" {
		return first
	}
	return second
}

// PreferFirstRune returns first non-empty rune
func PreferFirstRune(first rune, second rune) rune {
	if first != 0 {
		return first
	}
	return second
}

// Contains checks whether []string Contains string
func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
