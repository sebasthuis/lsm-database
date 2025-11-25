package database

import "errors"

var ErrKeyNotFound = errors.New("key not found")

type Database struct {
	memTable  *MemTable
	directory string
}

func Create(directory string) (*Database, error) {
	return &Database{
		memTable:  NewMemTable(),
		directory: directory,
	}, nil
}

func (db *Database) Put(key, value string) error {
	db.memTable.Put(key, value)
	return nil
}

func (db *Database) Get(key string) (string, error) {
	value, exists := db.memTable.Get(key)
	if !exists {
		return "", ErrKeyNotFound
	} else {
		return value, nil
	}
}

func (db *Database) Delete(key string) error {
	db.memTable.Delete(key)
	return nil
}
