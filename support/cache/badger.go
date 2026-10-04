package cache

import badger "github.com/dgraph-io/badger/v4"

type (
	badgerCache struct {
		db *badger.DB
	}
)

var _ Cache = (*badgerCache)(nil)

func NewBadgerCache(db *badger.DB) (Cache, error) {
	return &badgerCache{db: db}, nil
}

func (c *badgerCache) Get(key string) ([]byte, bool) {
	var value []byte
	err := c.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		value, err = item.ValueCopy(nil)
		return err
	})
	if err != nil {
		return nil, false
	}
	return value, true
}

func (c *badgerCache) Set(key string, value []byte) error {
	return c.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), value)
	})
}

func (c *badgerCache) Delete(key string) error {
	return c.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}
