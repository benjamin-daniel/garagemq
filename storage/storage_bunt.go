package storage

import (
	"fmt"

	"github.com/tidwall/buntdb"
	"github.com/valinurovam/garagemq/interfaces"

	"time"
)

// BuntDB implements wrapper for BuntDB database
type BuntDB struct {
	db *buntdb.DB
}

// NewBuntDB returns new instance of BuntDB wrapper
func NewBuntDB(storagePath string) *BuntDB {
	storage := &BuntDB{}

	storagePath = fmt.Sprintf("%s/%s", storagePath, "db")
	var db, err = buntdb.Open(storagePath)
	if err != nil {
		panic(err)
	}

	db.SetConfig(buntdb.Config{
		SyncPolicy:         buntdb.Always,
		AutoShrinkDisabled: true,
	})

	storage.db = db
	go storage.runStorageGC()

	return storage
}

// ProcessBatch process batch of operations
func (storage *BuntDB) ProcessBatch(batch []*interfaces.Operation) (err error) {
	return storage.db.Update(func(tx *buntdb.Tx) error {
		for _, op := range batch {
			if op.Op == interfaces.OpSet {
				tx.Set(op.Key, string(op.Value), nil)
			}
			if op.Op == interfaces.OpDel {
				tx.Delete(op.Key)
			}
		}
		return nil
	})
}

// Close properly closes BuntDB database
func (storage *BuntDB) Close() error {
	return storage.db.Close()
}

// Set adds a key-value pair to the database
func (storage *BuntDB) Set(key string, value []byte) (err error) {
	return storage.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(key, string(value), nil)
		return err
	})
}

// Del deletes a key
func (storage *BuntDB) Del(key string) (err error) {
	return storage.db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(key)
		return err
	})
}

// Get returns value by key
func (storage *BuntDB) Get(key string) (value []byte, err error) {
	storage.db.View(func(tx *buntdb.Tx) error {
		data, err := tx.Get(key)
		if err != nil {
			return err
		}
		value = make([]byte, len(data))
		copy(value, data)
		return nil
	})
	return
}

// Iterate iterates over all keys
func (storage *BuntDB) Iterate(fn func(key []byte, value []byte)) {
	storage.db.View(func(tx *buntdb.Tx) error {
		err := tx.Ascend("", func(key, value string) bool {
			fn([]byte(key), []byte(value))
			return true
		})
		return err
	})
}

func (storage *BuntDB) runStorageGC() {
	timer := time.Tick(30 * time.Minute)
	for {
		select {
		case <-timer:
			storage.db.Shrink()
		}
	}
}
