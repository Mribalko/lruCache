package lrucache

import (
	"reflect"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestNew(t *testing.T) {

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()
		const cap = 10
		got, err := New(cap)

		if err != nil {
			t.Errorf("not expected error = %v", err)
		}
		if got.cap != cap {
			t.Errorf("invalid capacity got = %v, want = %v", got.cap, cap)
		}

		if got.items == nil {
			t.Error("map not initialised")
		}

		if got.queue == nil {
			t.Error("list not initialised")
		}
	})

	t.Run("happy path with ttl", func(t *testing.T) {
		t.Parallel()
		const (
			cap   = 8
			ttl   = 10 * time.Second
			ticks = 4
		)

		got, err := New(cap, WithTTL(ttl, ticks))

		if err != nil {
			t.Errorf("not expected error = %v", err)
		}

		if got.ttl != ttl {
			t.Errorf("invalid ttl value: got = %v, want = %v", got.ttl, ttl)
		}

		if got.cf == nil {
			t.Error("clear function not initialised")
		}

	})

	cases := []struct {
		name  string
		cap   int
		ttl   time.Duration
		ticks int
	}{
		{
			"negative capacity",
			-10,
			2,
			2,
		},
		{
			"zero capacity",
			0,
			2,
			2,
		},
		{
			"zero ttl",
			2,
			0,
			2,
		},
		{
			"negative ttl",
			2,
			-4,
			2,
		},
		{
			"ticks equals one",
			2,
			2,
			1,
		},
		{
			"ticks equals zero",
			2,
			2,
			0,
		},
		{
			"negative ticks",
			2,
			2,
			-4,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := New(tt.cap, WithTTL(tt.ttl, tt.ticks))
			if err == nil {
				t.Errorf("error expected")
			}
		})
	}

}

func TestWithTTL(t *testing.T) {
	t.Run("number of cleans per ttl period", func(t *testing.T) {
		t.Parallel()
		const (
			ttl   = 100 * time.Millisecond
			ticks = 4
		)
		var executions int64

		lruCache := &LRUCache{
			cf: func(l *LRUCache) {
				atomic.AddInt64(&executions, 1)
			},
		}

		WithTTL(ttl, ticks)(lruCache)
		time.Sleep(ttl)
		lruCache.cancel()

		if got, want := int(atomic.LoadInt64(&executions)), ticks; got < want {
			t.Errorf("got = %d must be greater or equal than want = %d", got, want)
		}
	})

	t.Run("delete expired", func(t *testing.T) {
		t.Parallel()

		const (
			cap   = 2
			ttl   = 100 * time.Millisecond
			ticks = 4
		)

		cache, _ := New(cap, WithTTL(ttl, ticks))

		for i := range 4 {
			cache.Set(Key(strconv.Itoa(i)), i)
		}
		time.Sleep(ttl * 2)
		cache.cancel()

		cache.mu.Lock()
		defer cache.mu.Unlock()

		if got := len(cache.items); got != 0 {
			t.Errorf("cache.items isn't empty %v", cache.items)
		}

		if got := cache.queue.Len(); got != 0 {
			t.Errorf("cache.queue isn't empty %v", cache.queue)
		}

	})

}

func TestSet(t *testing.T) {

	t.Run("add one", func(t *testing.T) {
		t.Parallel()
		test := listItem{key: "one", value: 1}
		const (
			cap   = 2
			ttl   = 20 * time.Second
			ticks = 2
		)

		cache, _ := New(cap, WithTTL(ttl, ticks))
		cache.cancel()
		res := cache.Set(test.key, test.value)

		if res {
			t.Error("added new value: false expected")
		}

		if _, exist := cache.items[test.key]; !exist {
			t.Errorf("element wasn't added to cache.items = %v", cache.items)
		}

		got := cache.queue.Front().Value.(*listItem)
		if got.key != test.key || got.value != test.value {
			t.Errorf("element wasn't added to cache.queue: got = %v; want = %v", got, test)
		}

		if got := time.Until(got.expiresAt); got <= 0 {
			t.Errorf("expected positive value got = %v", got)

		}

	})

	t.Run("update item", func(t *testing.T) {
		t.Parallel()

		origTest := listItem{key: "one", value: 1}
		newTest := listItem{key: "one", value: "ONE"}
		const (
			cap   = 2
			ttl   = 20 * time.Second
			ticks = 2
		)

		cache, _ := New(cap, WithTTL(ttl, ticks))
		cache.cancel()

		cache.Set(origTest.key, origTest.value)
		origExpTime := cache.queue.Front().Value.(*listItem).expiresAt

		cache.Set("dummy", "dummy")

		res := cache.Set(newTest.key, newTest.value)
		if !res {
			t.Error("updated existing item: true expected")
		}

		cacheItem := cache.queue.Front().Value.(*listItem)
		if got := cacheItem; got.key != newTest.key || got.value != newTest.value {
			t.Errorf("cache.queue wasn't updated: got = %v; want = %v", got, newTest)
		}

		newExpTime := cacheItem.expiresAt
		if newExpTime.Sub(origExpTime) <= 0 {
			t.Errorf("expiresAt field wasn't updated: origValue = %v, newValue = %v", origExpTime, newExpTime)
		}

		if got, want := cache.items[origTest.key].Value.(*listItem).value, newTest.value; got != want {
			t.Errorf("cache.items wasn't updated: got = %v, want = %v", got, want)
		}

	})

	t.Run("overflow", func(t *testing.T) {
		t.Parallel()
		items := []listItem{
			{
				key: "one", value: 1,
			},
			{
				key: "two", value: 2,
			},
			{
				key: "three", value: 3,
			},
		}

		const cap = 2
		cache, _ := New(cap)

		for _, v := range items {
			cache.Set(v.key, v.value)
		}

		if _, exist := cache.items[items[0].key]; exist {
			t.Errorf("cache.items: key %v must be deleted", items[0].key)
		}

		if got, want := cache.queue.Front().Value, &items[2]; !reflect.DeepEqual(got, want) {
			t.Errorf("cache.queue wasn't updated: got = %v; want = %v", got, want)
		}

		if got, want := cache.queue.Len(), cap; got != want {
			t.Errorf("cache.queue is oversized: got = %v; want = %v", got, want)
		}

	})

}

func TestGet(t *testing.T) {
	t.Run("get existing", func(t *testing.T) {
		t.Parallel()

		const (
			cap   = 2
			ttl   = 20 * time.Second
			ticks = 2
		)

		cache, _ := New(cap, WithTTL(ttl, ticks))
		cache.cancel()

		test := listItem{key: "one", value: 1}

		cache.Set(test.key, test.value)
		origExpTime := cache.queue.Front().Value.(*listItem).expiresAt

		value, exist := cache.Get(test.key)
		newExpTime := cache.queue.Front().Value.(*listItem).expiresAt

		if newExpTime.Sub(origExpTime) <= 0 {
			t.Errorf("expiresAt field wasn't updated: origValue = %v, newValue = %v", origExpTime, newExpTime)
		}

		if got, want := value, test.value; !reflect.DeepEqual(got, want) {
			t.Errorf("items not equal got = %v, want = %v", got, want)
		}

		if !exist {
			t.Error("true expected")
		}

	})

	t.Run("get not existing", func(t *testing.T) {
		t.Parallel()

		const (
			cap   = 2
			neKey = "test"
		)
		test := listItem{key: "one", value: 1}

		cache, _ := New(cap)

		cache.Set(test.key, test.value)

		value, exist := cache.Get(neKey)

		if value != nil {
			t.Errorf("got not existent value = %v", value)
		}

		if exist {
			t.Error("false expected")
		}

	})
}

func TestClear(t *testing.T) {
	t.Parallel()

	const cap = 2
	test := listItem{key: "one", value: 1}

	cache, _ := New(cap)

	cache.Set(test.key, test.value)
	cache.Set(test.key, test.value)

	cache.Clear()

	if len(cache.items) != 0 {
		t.Errorf("cache.items isn't empty %v", cache.items)
	}

	if cache.queue.Len() != 0 {
		t.Error("cache.queue isn't empty")
	}

}
