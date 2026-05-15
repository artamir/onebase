package widget

import (
	"testing"
	"time"
)

func TestCache_GetPut(t *testing.T) {
	c := NewCache(time.Minute)
	if _, ok := c.get("missing"); ok {
		t.Fatal("empty cache returned hit")
	}
	c.put("a", Result{Name: "a", Title: "T"})
	got, ok := c.get("a")
	if !ok || got.Name != "a" {
		t.Fatalf("get after put: ok=%v name=%q", ok, got.Name)
	}
}

func TestCache_Expiry(t *testing.T) {
	c := NewCache(50 * time.Millisecond)
	c.put("k", Result{Name: "k"})
	if _, ok := c.get("k"); !ok {
		t.Fatal("fresh entry should be cached")
	}
	time.Sleep(80 * time.Millisecond)
	if _, ok := c.get("k"); ok {
		t.Fatal("expired entry should be evicted")
	}
}

func TestCache_Invalidate(t *testing.T) {
	c := NewCache(time.Minute)
	c.put("a", Result{Name: "a"})
	c.put("b", Result{Name: "b"})
	c.Invalidate()
	if _, ok := c.get("a"); ok {
		t.Fatal("Invalidate did not drop entries")
	}
}

func TestCache_NilSafe(t *testing.T) {
	var c *Cache
	if _, ok := c.get("x"); ok {
		t.Fatal("nil cache should miss")
	}
	c.put("x", Result{})
	c.Invalidate()
}

func TestCacheKey(t *testing.T) {
	if cacheKey("A", "u1") == cacheKey("A", "u2") {
		t.Fatal("different users must produce different keys")
	}
	if cacheKey("A", "u1") != cacheKey("A", "u1") {
		t.Fatal("same inputs must produce same key")
	}
}
