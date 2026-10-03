package ui

import (
	"testing"

	"github.com/eurico-martins/lazytree/internal/model"
)

func TestDiffCacheEvictsLeastRecentlyUsed(t *testing.T) {
	c := newDiffCache(2)
	f := []model.DiffFile{{NewPath: "x"}}
	c.put("a", f)
	c.put("b", f)
	if _, ok := c.get("a"); !ok { // touch a; b is now the oldest
		t.Fatal("a should be cached")
	}
	c.put("c", f)
	if _, ok := c.get("b"); ok {
		t.Error("b should have been evicted")
	}
	if _, ok := c.get("a"); !ok {
		t.Error("a should still be cached")
	}
	if _, ok := c.get("c"); !ok {
		t.Error("c should be cached")
	}
}
