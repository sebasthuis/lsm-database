package database

import "sync"

// TODO: Maintian sorted order in memtable

type MemTable struct {
	entries map[string]string
	lock    sync.RWMutex
}

func NewMemTable() *MemTable {
	return &MemTable{
		entries: make(map[string]string),
	}
}

func (m *MemTable) Put(key, value string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.entries[key] = value
}

func (m *MemTable) Get(key string) (string, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	value, exists := m.entries[key]
	return value, exists
}

func (m *MemTable) Delete(key string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.entries, key)
}
