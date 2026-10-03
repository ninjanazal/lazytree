package ui

import (
	"container/list"

	"github.com/eurico-martins/lazytree/internal/model"
)

// diffCacheSize bounds how many parsed diffs are kept. Commits are
// immutable, so entries never go stale; the bound only limits memory.
const diffCacheSize = 32

type diffCacheEntry struct {
	hash  string
	files []model.DiffFile
}

// diffCache is a small LRU of parsed diffs keyed by commit hash. It is a
// pointer inside AppModel (whose methods have value receivers) and is only
// touched from Update, so it needs no locking.
type diffCache struct {
	max   int
	order *list.List // front = most recently used
	items map[string]*list.Element
}

func newDiffCache(max int) *diffCache {
	return &diffCache{max: max, order: list.New(), items: make(map[string]*list.Element)}
}

func (c *diffCache) get(hash string) ([]model.DiffFile, bool) {
	el, ok := c.items[hash]
	if !ok {
		return nil, false
	}
	c.order.MoveToFront(el)
	return el.Value.(*diffCacheEntry).files, true
}

func (c *diffCache) put(hash string, files []model.DiffFile) {
	if el, ok := c.items[hash]; ok {
		el.Value.(*diffCacheEntry).files = files
		c.order.MoveToFront(el)
		return
	}
	c.items[hash] = c.order.PushFront(&diffCacheEntry{hash: hash, files: files})
	if c.order.Len() > c.max {
		oldest := c.order.Back()
		c.order.Remove(oldest)
		delete(c.items, oldest.Value.(*diffCacheEntry).hash)
	}
}
