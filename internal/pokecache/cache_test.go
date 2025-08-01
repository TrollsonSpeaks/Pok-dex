package pokecache

import (
    "fmt"
    "testing"
    "time"
)

func TestAddGet(t *testing.T) {
    const interval = 5 * time.Second
    cases := []struct {
        key string
        val []byte
    }{
        {
            key: "https://example.com",
            val: []byte("testdata"),
        },
        {
            key: "https://example.com/path",
            val: []byte("moretestdata"),
        },
    }

    for i, c := range cases {
        t.Run(fmt.Sprintf("Test case %v", i), func(t *testing.T) {
            cache := NewCache(interval)
            cache.Add(c.key, c.val)
            val, ok := cache.Get(c.key)
            if !ok {
                t.Errorf("expected to find key")
            }
            if string(val) != string(c.val) {
                t.Errorf("expected value %q, got %q", c.val, val)
            }
        })
    }
}

func TestReapLoop(t *testing.T) {
    const shortInterval = 5 * time.Millisecond
    cache := NewCache(shortInterval)
    cache.Add("https://example.com", []byte("testdata"))

    _, ok := cache.Get("https://example.com")
    if !ok {
        t.Errorf("expected to find key")
    }

    time.Sleep(10 * time.Millisecond)

    _, ok = cache.Get("https://example.com")
    if ok {
        t.Errorf("expected key to be reaped from cache")
    }
}
