// Package cache provides a byte-budgeted in-memory LRU, a Redis cache, and a
// two-level cache that satisfies thumbapp.Cache.
package cache

import (
	"container/list"
	"sync"

	"scout/internal/application/thumbapp"
)

// entry is one LRU node.
type entry struct {
	key   string
	value thumbapp.Thumbnail
	size  int64
}

// LRU is a concurrency-safe, byte-budgeted LRU cache for hot thumbnail bytes.
// The RWMutex protects only the in-memory cache state (not request concurrency).
type LRU struct {
	mu     sync.RWMutex
	budget int64
	used   int64
	ll     *list.List
	items  map[string]*list.Element
}

// NewLRU builds an LRU with the given byte budget.
func NewLRU(budget int64) *LRU {
	return &LRU{
		budget: budget,
		ll:     list.New(),
		items:  make(map[string]*list.Element),
	}
}

// Get returns a cached thumbnail and marks it most-recently-used.
func (c *LRU) Get(key string) (thumbapp.Thumbnail, bool) {
	c.mu.RLock()
	el, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return thumbapp.Thumbnail{}, false
	}
	c.mu.Lock()
	c.ll.MoveToFront(el)
	c.mu.Unlock()
	return el.Value.(*entry).value, true
}

// Set inserts or updates a thumbnail, evicting LRU entries to stay in budget.
func (c *LRU) Set(key string, t thumbapp.Thumbnail) {
	size := int64(len(t.Data))
	if size > c.budget {
		return // never cache an item larger than the whole budget
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.items[key]; ok {
		old := el.Value.(*entry)
		c.used += size - old.size
		old.value = t
		old.size = size
		c.ll.MoveToFront(el)
	} else {
		el := c.ll.PushFront(&entry{key: key, value: t, size: size})
		c.items[key] = el
		c.used += size
	}
	for c.used > c.budget {
		back := c.ll.Back()
		if back == nil {
			break
		}
		ev := back.Value.(*entry)
		c.ll.Remove(back)
		delete(c.items, ev.key)
		c.used -= ev.size
	}
}
