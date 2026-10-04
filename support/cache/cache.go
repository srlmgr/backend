package cache

type Cache interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte) error
	Delete(key string) error
}

type NoopCache struct{}

var _ Cache = (*NoopCache)(nil)

func NewNoopCache() Cache {
	c := NoopCache{}
	return &c
}

func (c *NoopCache) Get(key string) ([]byte, bool) {
	return nil, false
}

func (c *NoopCache) Set(key string, value []byte) error {
	return nil
}

func (c *NoopCache) Delete(key string) error {
	return nil
}
