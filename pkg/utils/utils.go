package utils

func Contains[T comparable](list []T, val T) bool {
	for _, elem := range list {
		if elem == val {
			return true
		}
	}
	return false
}

func Keys[T, U comparable](myMap map[T]U) []T {
	keys := make([]T, len(myMap))
	for k := range myMap {
		keys = append(keys, k)
	}
	return keys
}
