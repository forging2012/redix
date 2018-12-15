package main

import (
	"time"

	"github.com/dgraph-io/badger"
)

// DB - represents a DB structure
type DB struct {
	badger *badger.DB
}

// Set - sets a key with the specified value and optional ttl
func (db *DB) Set(k, v string, ttl int) error {
	return db.badger.Update(func(txn *badger.Txn) (err error) {
		if ttl < 1 {
			err = txn.Set([]byte(k), []byte(v))
		} else {
			err = txn.SetWithTTL([]byte(k), []byte(v), time.Duration(ttl)*time.Millisecond)
		}

		return err
	})
}

// MSet - sets multiple key-value pairs
func (db *DB) MSet(data map[string]string) error {
	return db.badger.Update(func(txn *badger.Txn) (err error) {
		for k, v := range data {
			txn.Set([]byte(k), []byte(v))
		}
		return nil
	})
}

// Get - fetches the value of the specified k
func (db *DB) Get(k string) (string, error) {
	var data string

	err := db.badger.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(k))
		if err != nil {
			return err
		}

		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		data = string(val)

		return nil
	})

	return data, err
}

// MGet - fetch multiple values of the specified keys
func (db *DB) MGet(keys []string) (data []string) {
	db.badger.View(func(txn *badger.Txn) error {
		for _, key := range keys {
			item, err := txn.Get([]byte(key))
			if err != nil {
				data = append(data, "")
				continue
			}
			val, err := item.ValueCopy(nil)
			if err != nil {
				data = append(data, "")
				continue
			}
			data = append(data, string(val))
		}
		return nil
	})

	return data
}

// Del - removes key(s) from the store
func (db *DB) Del(keys []string) error {
	return db.badger.Update(func(txn *badger.Txn) error {
		for _, key := range keys {
			txn.Delete([]byte(key))
		}

		return nil
	})
}

// Scan - iterate over the whole store using the handler function
func (db *DB) Scan(scannerOpt ScannerOptions) error {
	return db.badger.View(func(txn *badger.Txn) error {
		iteratorOpts := badger.DefaultIteratorOptions
		iteratorOpts.PrefetchValues = scannerOpt.FetchValues

		it := txn.NewIterator(iteratorOpts)
		defer it.Close()

		start := func(it *badger.Iterator) {
			if scannerOpt.Offset == "" {
				it.Rewind()
			} else {
				it.Seek([]byte(scannerOpt.Offset))
				if !scannerOpt.IncludeOffset && it.Valid() {
					it.Next()
				}
			}
		}

		valid := func(it *badger.Iterator) bool {
			if !it.Valid() {
				return false
			}

			if scannerOpt.Prefix != "" && !it.ValidForPrefix([]byte(scannerOpt.Prefix)) {
				return false
			}

			return true
		}

		for start(it); valid(it); it.Next() {
			var k, v []byte

			item := it.Item()
			k = item.KeyCopy(nil)

			if scannerOpt.FetchValues {
				v, _ = item.ValueCopy(nil)
			}

			if !scannerOpt.Handler(string(k), string(v)) {
				break
			}
		}

		return nil
	})
}
