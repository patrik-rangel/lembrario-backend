package api

func ptr[T any](v T) *T {
	return &v
}
