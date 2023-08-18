package shared

type Storage interface {
	Put(key string, value int64) error
	Get(key string) (int64, error)
}
