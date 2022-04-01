package tools

func Contains[T string](haystack []T, needle T) bool {
	for _, a := range haystack {
		if a == needle {
			return true
		}
	}
	return false
}
