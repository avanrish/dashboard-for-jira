package jira

import (
	"encoding/json"
	"sync"
	"time"
)

// RequestOptions changes how the client makes one request. The zero value always asks Jira.
type RequestOptions struct {
	// CacheTTL is how old a reused response may be, and how long the client keeps the response this request gets.
	// There is no default: zero always asks Jira, so a request gets older data only when it says how old.
	CacheTTL time.Duration
}

// cached decodes into out the response stored under key if it is younger than the request's TTL.
// Otherwise it decodes the response that load gets from Jira, and keeps it for that TTL.
func (c *Client) cached(key string, options RequestOptions, out any, load func() ([]byte, error)) error {
	if body, ok := c.cache.get(key, options.CacheTTL); ok {
		return json.Unmarshal(body, out)
	}

	body, err := load()
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return err
	}
	c.cache.put(key, body, options.CacheTTL)
	return nil
}

// responseCache keeps Jira's successful responses in memory, keyed by request URL.
type responseCache struct {
	mu      sync.Mutex
	entries map[string]cacheEntry
	now     func() time.Time // replaced in tests
}

type cacheEntry struct {
	body    []byte
	fetched time.Time
	expires time.Time // when put drops the entry, which bounds memory use
}

// get returns the response stored for key if it is younger than maxAge.
func (c *responseCache) get(key string, maxAge time.Duration) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok || c.now().Sub(entry.fetched) >= maxAge {
		return nil, false
	}
	return entry.body, true
}

// put stores a response just fetched by a request with the given TTL, replacing the older one, and drops expired entries.
func (c *responseCache) put(key string, body []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	for k, entry := range c.entries {
		if !now.Before(entry.expires) {
			delete(c.entries, k)
		}
	}
	if c.entries == nil {
		c.entries = make(map[string]cacheEntry)
	}
	c.entries[key] = cacheEntry{body: body, fetched: now, expires: now.Add(ttl)}
}
