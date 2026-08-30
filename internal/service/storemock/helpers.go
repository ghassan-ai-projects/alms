package storemock

// containsSubstring is a case-insensitive substring check.
func containsSubstring(s, substr string) bool {
	if substr == "" {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			sc := s[i+j]
			tc := substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if tc >= 'A' && tc <= 'Z' {
				tc += 32
			}
			if sc != tc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// tagsOverlap checks if any tag in a overlaps with any tag in b.
func tagsOverlap(a, b []string) bool {
	for _, ta := range a {
		for _, tb := range b {
			if ta == tb {
				return true
			}
		}
	}
	return false
}
