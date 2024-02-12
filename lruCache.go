package lruCache

import (
	"container/list"
	"errors"
)

type Key string

type LRUCache struct {
	// cache capacity
	cap int
	// hash table
	items map[Key]*list.Element
	// order list
	queue *list.List
}

type listItem struct {
	key   Key
	value any
}

// Creates new LRUCache
func New(cap int) (*LRUCache, error) {
	if cap <= 0 {
		return nil, errors.New("cap must be positive")
	}
	res := &LRUCache{
		cap:   cap,
		items: make(map[Key]*list.Element, cap),
		queue: list.New(),
	}
	return res, nil
}

// Adds value to cache.
// Return: true - existing element was updated, false - new element was added
func (l *LRUCache) Set(key Key, value any) bool {
	if node, exist := l.items[key]; exist {
		node.Value.(*listItem).value = value
		l.queue.MoveToFront(node)
		return true
	}

	if len(l.items) == l.cap {
		lastNode := l.queue.Back()
		item := lastNode.Value.(*listItem)
		l.queue.Remove(lastNode)
		delete(l.items, item.key)
	}

	item := &listItem{key, value}
	l.items[key] = l.queue.PushFront(item)
	return false
}

// Gets value from cache
// Return: true - element exists, false - element doesn't exist
func (l *LRUCache) Get(key Key) (any, bool) {
	node, exist := l.items[key]
	if !exist {
		return nil, false
	}

	l.queue.MoveToFront(node)
	return node.Value.(*listItem).value, true
}

// Clears cache
func (l *LRUCache) Clear() {
	clear(l.items)
	l.queue.Init()
}
