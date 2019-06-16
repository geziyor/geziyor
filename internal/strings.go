package internal

// PreferFirst returns first non-empty string
func PreferFirst(first string, second string) string {
	if first != "" {
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
