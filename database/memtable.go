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
)

type node struct {
	key   string
	value string
	// Tower, forward pointers for each level. Higher is more sparse
	next []*node
}
type MemTable struct {
	head   *node
	level  int
	random *rand.Rand
	lock   sync.RWMutex
}

func NewMemTable() *MemTable {
	return &MemTable{
		head:   &node{next: make([]*node, maxLevel)},
		level:  1,
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (m *MemTable) Put(key, value string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	previous := make([]*node, maxLevel)

	current := m.head
	for i := m.level - 1; i >= 0; i-- {
		for current.next[i] != nil && current.next[i].key < key {
			current = current.next[i]
		}
		previous[i] = current
	}

	current = current.next[0]

	if current != nil && current.key == key {
		current.value = value
		return
	}

	newLevel := m.randomLevel()

	if newLevel > m.level {
		for i := m.level; i < newLevel; i++ {
			previous[i] = m.head
		}
		m.level = newLevel
	}

	newNode := &node{
		key:   key,
		value: value,
		next:  make([]*node, newLevel),
	}

	for i := range newLevel {
		newNode.next[i] = previous[i].next[i]
		previous[i].next[i] = newNode
	}

}

func (m *MemTable) Get(key string) (string, bool) {
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
		return current.value, true
	} else {
		return "", false
	}
}

func (m *MemTable) Delete(key string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	previous := make([]*node, maxLevel)
	current := m.head

	for i := m.level - 1; i >= 0; i-- {
		for current.next[i] != nil && current.next[i].key < key {
			current = current.next[i]
		}
		previous[i] = current
	}

	current = current.next[0]

	if current != nil && current.key == key {
		for i := 0; i < m.level; i++ {
			if previous[i].next[i] != current {
				break
			}
			previous[i].next[i] = current.next[i]
		}
	}

	for m.level > 1 && m.head.next[m.level-1] == nil {
		m.level--
	}
}

func (m *MemTable) randomLevel() int {
	level := 1
	for m.random.Float64() < promoteProbability && level < maxLevel {
		level++
	}
	return level
}
