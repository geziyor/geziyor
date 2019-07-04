package internal

// DefaultString returns first non-empty string
func DefaultString(val string, valDefault string) string {
	if val != "" {
		return val
	}
	return valDefault
}

// DefaultRune returns first non-empty rune
func DefaultRune(val rune, valDefault rune) rune {
	if val != 0 {
		return val
	}
	return valDefault
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
