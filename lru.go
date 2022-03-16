// Copyright 2020 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package tinylru

import "sync"

// DefaultSize is the default maximum size of an LRU cache before older items
// get automatically evicted.
const DefaultSize = 256

type lruItem[Key comparable, Value any] struct {
	key   Key                  // user-defined key
	value Value                // user-defined value
	prev  *lruItem[Key, Value] // prev item in list. More recently used
	next  *lruItem[Key, Value] // next item in list. Less recently used
}

// LRU implements an LRU cache
type LRU[Key comparable, Value any] struct {
	mu    sync.RWMutex                 // protect all things
	size  int                          // max number of items.
	items map[Key]*lruItem[Key, Value] // active items
	head  *lruItem[Key, Value]         // head of list
	tail  *lruItem[Key, Value]         // tail of list
}

//go:noinline
func (lru *LRU[Key, Value]) init() {
	lru.items = make(map[Key]*lruItem[Key, Value])
	lru.head = new(lruItem[Key, Value])
	lru.tail = new(lruItem[Key, Value])
	lru.head.next = lru.tail
	lru.tail.prev = lru.head
	if lru.size == 0 {
		lru.size = DefaultSize
	}
}

func (lru *LRU[Key, Value]) evict() *lruItem[Key, Value] {
	item := lru.tail.prev
	lru.pop(item)
	delete(lru.items, item.key)
	return item
}

func (lru *LRU[Key, Value]) pop(item *lruItem[Key, Value]) {
	item.prev.next = item.next
	item.next.prev = item.prev
}

func (lru *LRU[Key, Value]) push(item *lruItem[Key, Value]) {
	lru.head.next.prev = item
	item.next = lru.head.next
	item.prev = lru.head
	lru.head.next = item
}

// Resize sets the maximum size of an LRU cache. If this value is less than
// the number of items currently in the cache, then items will be evicted.
// Returns evicted items.
// This operation will panic if the size is less than one.
func (lru *LRU[Key, Value]) Resize(size int) (evictedKeys []Key,
	evictedValues []Value) {
	if size <= 0 {
		panic("invalid size")
	}

	lru.mu.Lock()
	defer lru.mu.Unlock()
	for size < len(lru.items) {
		item := lru.evict()
		evictedKeys = append(evictedKeys, item.key)
		evictedValues = append(evictedValues, item.value)
	}
	lru.size = size
	return evictedKeys, evictedValues
}

// Len returns the length of the lru cache
func (lru *LRU[Key, Value]) Len() int {
	lru.mu.RLock()
	defer lru.mu.RUnlock()
	return len(lru.items)
}

// SetEvicted sets or replaces a value for a key. If this operation causes an
// eviction then the evicted item is returned.
func (lru *LRU[Key, Value]) SetEvicted(key Key, value Value) (
	prev Value, replaced bool, evictedKey Key,
	evictedValue Value, evicted bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	if lru.items == nil {
		lru.init()
	}
	item := lru.items[key]
	if item == nil {
		if len(lru.items) == lru.size {
			item = lru.evict()
			evictedKey, evictedValue, evicted = item.key, item.value, true
		} else {
			item = new(lruItem[Key, Value])
		}
		item.key = key
		item.value = value
		lru.push(item)
		lru.items[key] = item
	} else {
		prev, replaced = item.value, true
		item.value = value
		if lru.head.next != item {
			lru.pop(item)
			lru.push(item)
		}
	}
	return prev, replaced, evictedKey, evictedValue, evicted
}

// Set or replace a value for a key.
func (lru *LRU[Key, Value]) Set(key Key, value Value) (prev Value,
	replaced bool) {
	prev, replaced, _, _, _ = lru.SetEvicted(key, value)
	return prev, replaced
}

// Get a value for key
func (lru *LRU[Key, Value]) Get(key Key) (value Value, ok bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	item := lru.items[key]
	if item == nil {
    return
	}
	if lru.head.next != item {
		lru.pop(item)
		lru.push(item)
	}
	return item.value, true
}

// Contains returns true if the key exists.
func (lru *LRU[Key, Value]) Contains(key Key) bool {
	lru.mu.RLock()
	defer lru.mu.RUnlock()
	_, ok := lru.items[key]
	return ok
}

// Peek returns the value for key value without updating
// the recently used status.
func (lru *LRU[Key, Value]) Peek(key Key) (value Value, ok bool) {
	lru.mu.RLock()
	defer lru.mu.RUnlock()

	if item := lru.items[key]; item != nil {
		return item.value, true
	}
  return
}

// Delete a value for a key
func (lru *LRU[Key, Value]) Delete(key Key) (prev Value, deleted bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	item := lru.items[key]
	if item == nil {
    return
	}
	delete(lru.items, key)
	lru.pop(item)
	return item.value, true
}

// Range iterates over all key/values in the order of most recently to
// least recently used items.
// It's not safe to call other LRU operations while ranging.
func (lru *LRU[Key, Value]) Range(iter func(key Key, value Value) bool) {
	lru.mu.RLock()
	defer lru.mu.RUnlock()
	if head := lru.head; head != nil {
		item := head.next
		for item != lru.tail {
			if !iter(item.key, item.value) {
				return
			}
			item = item.next
		}
	}
}

// Reverse iterates over all key/values in the order of least recently to
// most recently used items.
// It's not safe to call other LRU operations while ranging.
func (lru *LRU[Key, Value]) Reverse(iter func(key Key, value Value) bool) {
	lru.mu.RLock()
	defer lru.mu.RUnlock()
	if tail := lru.tail; tail != nil {
		item := tail.prev
		for item != lru.head {
			if !iter(item.key, item.value) {
				return
			}
			item = item.prev
		}
	}
}
