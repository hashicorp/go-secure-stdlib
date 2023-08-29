package shared

type Counter interface {
	Increment(key string, value int64, storage Storage) (int64, error)
}
