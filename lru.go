package lrucache

import (
	"container/list"
	"context"
	"errors"
	"sync"
	"time"
)

type (
	Key         string
	Option      func(*LRUCache) error
	cleanerFunc func(*LRUCache)
)

type LRUCache struct {
	cap    int           // cache capacity
	ttl    time.Duration // ttl
	cancel context.CancelFunc
	cf     cleanerFunc
	mu     sync.Mutex
	items  map[Key]*list.Element // hash table
	queue  *list.List            // order list
}

type listItem struct {
	key       Key
	value     any
	expiresAt time.Time
}

// Creates new LRUCache
func New(cap int, options ...Option) (*LRUCache, error) {
	if cap <= 0 {
		return nil, errors.New("cap must be positive")
	}
	lruCache := &LRUCache{
		cap:   cap,
		items: make(map[Key]*list.Element, cap),
		queue: list.New(),
		cf:    clearExpired,
	}

	for _, opt := range options {
		if err := opt(lruCache); err != nil {
			return nil, err
		}
	}
	return lruCache, nil
}

// Sets time-to-live option.
// Ticks (must be greater 1) - the number of checks for expired elements during the ttl period.
func WithTTL(ttl time.Duration, ticks int) Option {
	return func(l *LRUCache) error {
		if ttl <= 0 {
			return errors.New("ttl duration must be positive")
		}

		if ticks <= 1 {
			return errors.New("ticks must be greater 1")
		}

		l.ttl = ttl

		ctx, cancel := context.WithCancel(context.Background())
		l.cancel = cancel

		go func() {

			ticker := time.NewTicker(l.ttl / time.Duration(ticks))
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					l.cf(l)
				}
			}
		}()

		return nil
	}
}

// Adds value to cache.
// Return: true - existing element was updated, false - new element was added
func (l *LRUCache) Set(key Key, value any) bool {

	newItem := &listItem{key: key, value: value}
	if l.ttl > 0 {
		newItem.expiresAt = time.Now().Add(l.ttl)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if node, exist := l.items[key]; exist {
		node.Value = newItem
		l.queue.MoveToFront(node)
		return true
	}

	if len(l.items) == l.cap {
		l.deleteItem(l.queue.Back())
	}

	l.items[key] = l.queue.PushFront(newItem)
	return false
}

// Gets value from cache
// Return: true - element exists, false - element doesn't exist
func (l *LRUCache) Get(key Key) (any, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	node, exist := l.items[key]
	if !exist {
		return nil, false
	}

	li := node.Value.(*listItem)
	if l.ttl > 0 {
		li.expiresAt = time.Now().Add(l.ttl)
	}
	l.queue.MoveToFront(node)
	return li.value, true
}

// Clears cache, cancels ttl checks
func (l *LRUCache) Clear() {
	if l.cancel != nil {
		l.cancel()
		l.ttl = 0
	}

	clear(l.items)
	l.queue.Init()
}

// Clears expired cache items
func clearExpired(l *LRUCache) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.queue.Len() == 0 {
		return
	}

	for node := l.queue.Back(); node != nil; {
		if expiresAt := node.Value.(*listItem).expiresAt; time.Until(expiresAt) > 0 {
			return
		}

		delNode := node
		node = node.Prev()
		l.deleteItem(delNode)

	}
}

// Deletes node from the queue and the hashtable
func (l *LRUCache) deleteItem(node *list.Element) {
	l.queue.Remove(node)
	item := node.Value.(*listItem)
	delete(l.items, item.key)
}
