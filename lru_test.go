package lrucache

import (
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()
		const cap = 10
		got, err := New(cap)

		if err != nil {
			t.Errorf("New() not expected error = %v", err)
			return
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

	t.Run("negative capacity", func(t *testing.T) {
		t.Parallel()
		const cap = -10
		_, err := New(cap)
		if err == nil {
			t.Error("error expected")
		}
	})

	t.Run("zero capacity", func(t *testing.T) {
		t.Parallel()
		const cap = 0
		_, err := New(cap)
		if err == nil {
			t.Error("error expected")
		}
	})

}

func TestSet(t *testing.T) {

	t.Run("add one", func(t *testing.T) {
		t.Parallel()
		test := listItem{"one", 1}
		const cap = 2

		cache, _ := New(cap)
		res := cache.Set(test.key, test.value)

		if res {
			t.Error("added new value: false expected")
		}

		if _, exist := cache.items[test.key]; !exist {
			t.Errorf("element wasn't added to cache.items = %v", cache.items)
		}

		if got, want := cache.queue.Front().Value, &test; !reflect.DeepEqual(got, want) {
			t.Errorf("element wasn't added to cache.queue: got = %v; want = %v", got, want)
		}

	})

	t.Run("update item", func(t *testing.T) {
		t.Parallel()

		origTest := listItem{"one", 1}
		newTest := listItem{"one", "ONE"}
		const cap = 2

		cache, _ := New(cap)
		cache.Set(origTest.key, origTest.value)
		cache.Set("dummy", "dummy")

		res := cache.Set(newTest.key, newTest.value)
		if !res {
			t.Error("updated existing item: true expected")
		}

		if got, want := cache.items[origTest.key].Value, &newTest; !reflect.DeepEqual(got, want) {
			t.Errorf("cache.items wasn't updated: got = %v, want = %v", got, want)
		}

		if got, want := cache.queue.Front().Value, &newTest; !reflect.DeepEqual(got, want) {
			t.Errorf("cache.queue wasn't updated: got = %v; want = %v", got, want)
		}
	})

	t.Run("overflow", func(t *testing.T) {
		t.Parallel()
		items := []listItem{
			{
				"one", 1,
			},
			{
				"two", 2,
			},
			{
				"three", 3,
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

		const cap = 2
		test := listItem{"one", 1}

		cache, _ := New(cap)

		cache.Set(test.key, test.value)

		value, exist := cache.Get(test.key)

		if got, want := value, test.value; !reflect.DeepEqual(got, want) {
			t.Errorf("items not equal got = %v, want = %v", got, want)
		}

		if !exist {
			t.Error("true expected")
		}

	})

	t.Run("get existing", func(t *testing.T) {
		t.Parallel()

		const (
			cap   = 2
			neKey = "test"
		)
		test := listItem{"one", 1}

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
	test := listItem{"one", 1}

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
