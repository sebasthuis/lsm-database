package database

import (
	"math/rand"
	"sync"
	"time"
)

const (
	// Efficient for up to 2^25 elements (~33 million) entries
	maxLevel           = 25
	promoteProbability = 0.5
	// Approximate memory overhead per node on 64-bit systems:
	// string (16) + string (16) + slice header (24) + bool (1 + 7 padding) = 64 bytes
	nodeOverhead = 64
	pointerSize  = 8
)

type node struct {
	key       string
	value     string
	tombstone bool
	// Tower, forward pointers for each level. Higher is more sparse
	next []*node
}
type MemTable struct {
	head   *node
	level  int
	size   int // Tracks total bytes (key + value)
	random *rand.Rand
	lock   sync.RWMutex
}

// TODO: Rename function to be consistent with SSTable or move into separate
// packages
func NewMemTable() *MemTable {
	return &MemTable{
		head:   &node{next: make([]*node, maxLevel)},
		level:  1,
		size:   nodeOverhead + (maxLevel * pointerSize),
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (m *MemTable) Put(key string, value string) int {
	m.lock.Lock()
	defer m.lock.Unlock()

	predecessors, target := m.traverse(key)

	if target != nil && target.key == key {
		m.size += calcSizeOnUpdate(target.value, value)
		target.value = value
		return m.size
	}

	newLevel := m.randomLevel()

	if newLevel > m.level {
		for i := m.level; i < newLevel; i++ {
			predecessors[i] = m.head
		}
		m.level = newLevel
	}

	newNode := &node{
		key:       key,
		value:     value,
		tombstone: false,
		next:      make([]*node, newLevel),
	}

	for i := range newLevel {
		newNode.next[i] = predecessors[i].next[i]
		predecessors[i].next[i] = newNode
	}

	m.size += calcSizeRow(key, value, newLevel)
	return m.size
}

func (m *MemTable) Get(key string) LookupResult {
	m.lock.RLock()
	defer m.lock.RUnlock()

	current := m.head
	for i := m.level - 1; i >= 0; i-- {
		for current.next[i] != nil && current.next[i].key < key {
			current = current.next[i]
		}
	}

	current = current.next[0]

	if current != nil && current.key == key {
		if current.tombstone {
			return LookupResult{Status: Deleted}
		}
		return LookupResult{Value: current.value, Status: Found}
	} else {
		return LookupResult{Status: NotFound}
	}
}

func (m *MemTable) Delete(key string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	_, target := m.traverse(key)

	if target != nil && target.key == key {
		m.size += calcSizeOnUpdate(target.value, "")
		target.value = ""
		target.tombstone = true
	}
}

func (m *MemTable) All() []Entry {
	m.lock.RLock()
	defer m.lock.RUnlock()

	entries := make([]Entry, 0)
	current := m.head.next[0]
	for current != nil {
		entries = append(entries, Entry{Key: current.key, Value: current.value, Tombstone: current.tombstone})
		current = current.next[0]
	}
	return entries
}

// Traverses the skiplist and returns previous node and candidate for the node
func (m *MemTable) traverse(key string) ([]*node, *node) {
	predecessors := make([]*node, maxLevel)
	current := m.head
	for i := m.level - 1; i >= 0; i-- {
		for current.next[i] != nil && current.next[i].key < key {
			current = current.next[i]
		}
		predecessors[i] = current
	}
	target := current.next[0]
	return predecessors, target
}

func (m *MemTable) randomLevel() int {
	level := 1
	for m.random.Float64() < promoteProbability && level < maxLevel {
		level++
	}
	return level
}

func calcSizeOnUpdate(old string, new string) int {
	return len(new) - len(old)
}

func calcSizeRow(key string, value string, level int) int {
	return len(key) + len(value) + nodeOverhead + (level * pointerSize)
}
