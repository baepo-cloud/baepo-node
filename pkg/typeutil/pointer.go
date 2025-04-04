package typeutil

func Ptr[T any](value T) *T {
	return &value
}

func PtrEqual[T comparable](a *T, b *T) bool {
	if (a == nil && b == nil) || (a != b) {
		return false
	}
	return *a == *b
}

func Deref[T any](value *T) T {
	if value == nil {
		defaultValue := new(T)
		return *defaultValue
	}
	return *value
}
