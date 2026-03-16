package client

import (
	"crypto/sha256"
	"encoding/hex"
	"text/template"

	lru "github.com/hashicorp/golang-lru/v2"
)

const (
	// DefaultCacheSize is the default maximum number of templates to cache
	DefaultCacheSize = 100
)

// TemplateCache implements an LRU cache for parsed templates using
// github.com/hashicorp/golang-lru which provides a thread-safe,
// size-bounded LRU cache with O(1) operations.
type TemplateCache struct {
	cache *lru.Cache[string, *template.Template]
}

// NewTemplateCache creates a new LRU cache with the specified maximum size
func NewTemplateCache(maxSize int) *TemplateCache {
	if maxSize <= 0 {
		maxSize = DefaultCacheSize
	}
	// lru.New only returns an error if size <= 0, which we've guarded against.
	cache, _ := lru.New[string, *template.Template](maxSize)
	return &TemplateCache{cache: cache}
}

// Get retrieves a template from the cache
func (c *TemplateCache) Get(key string) (*template.Template, bool) {
	return c.cache.Get(key)
}

// Put adds a template to the cache
func (c *TemplateCache) Put(key string, tmpl *template.Template) {
	c.cache.Add(key, tmpl)
}

// hashTemplate generates a cache key from the template string
func hashTemplate(tmplStr string) string {
	h := sha256.Sum256([]byte(tmplStr))
	return hex.EncodeToString(h[:])
}
