package tinylru

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func init() {
	seed := time.Now().UnixNano()
	println(seed)
	rand.Seed(seed)
}

type tItem struct {
	key interface{}
	val int
}

func TestLRU(t *testing.T) {
	N := DefaultSize * 10
	var items []tItem
	vals := rand.Perm(N)
	for i := 0; i < N; i++ {
		items = append(items, tItem{key: fmt.Sprint(vals[i]), val: vals[i]})
	}

	size := DefaultSize

	// Set items
	var cache LRU
	for i := 0; i < len(items); i++ {
		prev, replaced, evictedKey, evictedValue, evicted :=
			cache.SetEvicted(items[i].key, items[i].val)
		if replaced {
			t.Fatal("expected false")
		}
		if prev != nil {
			t.Fatal("expected nil")
		}
		if evicted {
			if i < size {
				t.Fatalf("evicted too soon: %d", i)
			}
			if evictedKey != items[i-size].key {
				t.Fatalf("expected %v, got %v",
					items[i-size].key, evictedKey)
			}
			if evictedValue != items[i-size].val {
				t.Fatalf("expected %v, got %v",
					items[i-size].val, evictedValue)
			}
		}
	}
	if cache.Len() != size {
		t.Fatalf("expected %v, got %v", size, cache.Len())
	}

	evictedKeys, evictedValues := cache.Resize(size / 2)
	if len(evictedKeys) != DefaultSize/2 {
		t.Fatalf("expected %v, got %v", DefaultSize/2, len(evictedKeys))
	}
	if len(evictedValues) != DefaultSize/2 {
		t.Fatalf("expected %v, got %v", DefaultSize/2, len(evictedValues))
	}
	for i := 0; i < len(evictedKeys); i++ {
		if evictedKeys[i] != items[len(items)-size+i].key {
			t.Fatalf("expected %v, got %v",
				items[len(items)-size+i].key, evictedKeys[i])
		}
		if evictedValues[i] != items[len(items)-size+i].val {
			t.Fatalf("expected %v, got %v",
				items[len(items)-size+i].val, evictedValues[i])
		}
	}
	size /= 2

	idx := size - 1
	res := make([]tItem, size)
	cache.Range(func(key, value interface{}) bool {
		res[idx] = tItem{key: key, val: value.(int)}
		idx--
		return true
	})
	for i, j := len(items)-size, 0; i < len(items); i, j = i+1, j+1 {
		if items[i] != res[j] {
			t.Fatal("mismatch")
		}
	}
	var recent tItem
	cache.Range(func(key, value interface{}) bool {
		recent = tItem{key: key, val: value.(int)}
		return false
	})
	if items[len(items)-1] != recent {
		t.Fatal("mismatch")
	}

	idx = size - 1
	res = make([]tItem, size)
	cache.Reverse(func(key, value interface{}) bool {
		res[idx] = tItem{key: key, val: value.(int)}
		idx--
		return true
	})
	for i, j := len(items)-1, 0; i >= len(items)-size; i, j = i-1, j+1 {
		if items[i] != res[j] {
			t.Fatal("mismatch")
		}
	}
	var least tItem
	cache.Reverse(func(key, value interface{}) bool {
		least = tItem{key: key, val: value.(int)}
		return false
	})
	if items[len(items)-size] != least {
		t.Fatal("mismatch")
	}

	// Get items
	for i := 0; i < len(items); i++ {
		value, ok := cache.Get(items[i].key)
		if i < len(items)-size {
			if ok {
				t.Fatal("expected false")
			}
			if value != nil {
				t.Fatal("expected nil")
			}
		} else {
			if !ok {
				t.Fatal("expected true")
			}
			if value != items[i].val {
				t.Fatalf("expected %v, got %v",
					items[i].val, value)
			}
		}
	}

	// Overwrite the last items
	for i := len(items) - size; i < len(items); i++ {
		tprev := items[i].val
		items[i].val = tprev + 1
		prev, replaced, _, _, evicted :=
			cache.SetEvicted(items[i].key, items[i].val)
		if !replaced {
			t.Fatal("expected true")
		}
		if prev != tprev {
			t.Fatalf("expected %v, got %v",
				tprev, prev)
		}
		if evicted {
			t.Fatalf("expected false")
		}
	}

	for i := len(items) - size; i < len(items); i++ {
		prev, deleted := cache.Delete(items[i].key)
		if !deleted {
			t.Fatal("expected true")
		}
		if prev != items[i].val {
			t.Fatalf("expected %v, got %v",
				items[i].val, prev)

		}
	}

	func() {
		defer func() {
			s, ok := recover().(string)
			if !ok || s != "invalid size" {
				t.Fatalf("expected '%v', got '%v'", "invalid size", s)
			}

		}()
		cache.Resize(0)
	}()

	prev, deleted := cache.Delete("hello")
	if deleted {
		t.Fatal("expected false")
	}
	if prev != nil {
		t.Fatal("expected nil")
	}

	prev, replaced := cache.Set("hello", "world")
	if replaced {
		t.Fatal("expected false")
	}
	if prev != nil {
		t.Fatal("expected nil")
	}
}

func BenchmarkSet(b *testing.B) {
	items := make([]tItem, b.N)
	for i := 0; i < b.N; i++ {
		items[i] = tItem{key: fmt.Sprint(rand.Int())}
	}
	b.ResetTimer()
	b.ReportAllocs()
	var cache LRU
	for i := 0; i < b.N; i++ {
		cache.Set(items[i].key, items[i].val)
	}
}

func TestLRUInt(t *testing.T) {
	var cache LRU
	cache.Set(123, 123)
	cache.Set(123, 456)
	v, _ := cache.Get(123)
	if v.(int) != 456 {
		t.Fatalf("expected %v, got %v", 456, v)
	}
}
