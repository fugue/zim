package definitions

func mergeStr(a, b string) string {
	if b != "" {
		return b
	}
	return a
}

func mergeInt(a, b int) int {
	if b != 0 {
		return b
	}
	return a
}

func mergeBool(a, b bool) bool {
	if b {
		return true
	}
	return a
}

func copyStrings(input []string) []string {
	if input == nil {
		return nil
	}
	result := make([]string, len(input))
	copy(result, input)
	return result
}

func mergeStrings(a, b []string) []string {
	if len(b) > 0 {
		return copyStrings(b)
	}
	return copyStrings(a)
}

func copyStringsMap(m map[string]string) map[string]string {
	result := map[string]string{}
	for k, v := range m {
		result[k] = v
	}
	return result
}

func mergeStringsMap(a, b map[string]string) map[string]string {
	result := copyStringsMap(a)
	for k, v := range b {
		result[k] = v
	}
	return result
}
