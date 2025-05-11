package typeutil

func Includes[T comparable](slice []T, expected T) bool {
	for _, item := range slice {
		if item == expected {
			return true
		}
	}

	return false
}
