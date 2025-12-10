package client

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"text/template"
)

const (
	// DefaultCacheSize is the default maximum number of templates to cache
	DefaultCacheSize = 100
)

// cacheEntry represents a cached template with its key
type cacheEntry struct {
	key  string
	tmpl *template.Template
}

// TemplateCache implements an LRU cache for parsed templates
type TemplateCache struct {
	cache   map[string]*list.Element // map[hash]*list.Element
	lruList *list.List               // LRU list of *cacheEntry
	maxSize int
	mu      sync.Mutex
}

// NewTemplateCache creates a new LRU cache with the specified maximum size
func NewTemplateCache(maxSize int) *TemplateCache {
	if maxSize <= 0 {
		maxSize = DefaultCacheSize
	}
	return &TemplateCache{
		cache:   make(map[string]*list.Element),
		lruList: list.New(),
		maxSize: maxSize,
	}
}

// Get retrieves a template from the cache
func (c *TemplateCache) Get(key string) (*template.Template, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, found := c.cache[key]; found {
		// Move to front (most recently used)
		c.lruList.MoveToFront(elem)
		return elem.Value.(*cacheEntry).tmpl, true
	}
	return nil, false
}

// Put adds a template to the cache
func (c *TemplateCache) Put(key string, tmpl *template.Template) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if key already exists
	if elem, found := c.cache[key]; found {
		// Update existing entry and move to front
		c.lruList.MoveToFront(elem)
		elem.Value.(*cacheEntry).tmpl = tmpl
		return
	}

	// Add new entry
	entry := &cacheEntry{
		key:  key,
		tmpl: tmpl,
	}
	elem := c.lruList.PushFront(entry)
	c.cache[key] = elem

	// Evict oldest entry if cache is full
	if c.lruList.Len() > c.maxSize {
		c.evictOldest()
	}
}

// evictOldest removes the least recently used entry from the cache
func (c *TemplateCache) evictOldest() {
	oldest := c.lruList.Back()
	if oldest != nil {
		c.lruList.Remove(oldest)
		oldEntry := oldest.Value.(*cacheEntry)
		delete(c.cache, oldEntry.key)
	}
}

// Stats returns current cache statistics
func (c *TemplateCache) Stats() (size, maxSize int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lruList.Len(), c.maxSize
}

// Clear removes all entries from the cache
func (c *TemplateCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*list.Element)
	c.lruList.Init()
}

// Len returns the current number of entries in the cache
func (c *TemplateCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lruList.Len()
}

// hashTemplate generates a cache key from the template string
func hashTemplate(tmplStr string) string {
	h := sha256.Sum256([]byte(tmplStr))
	return hex.EncodeToString(h[:])
}
