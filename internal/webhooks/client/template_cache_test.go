package client

import (
	"fmt"
	"sync"
	"testing"
	"text/template"
)

func TestTemplateCacheBasicOperations(t *testing.T) {
	cache := NewTemplateCache(3)
	tmpl1 := template.Must(template.New("test1").Parse("Hello {{.}}"))
	cache.Put("key1", tmpl1)

	retrieved, found := cache.Get("key1")
	if !found {
		t.Error("Expected to find key1 in cache")
	}
	if retrieved != tmpl1 {
		t.Error("Retrieved template does not match original")
	}

	_, found = cache.Get("nonexistent")
	if found {
		t.Error("Expected not to find nonexistent key")
	}
}

func TestTemplateCacheLRUEviction(t *testing.T) {
	cache := NewTemplateCache(2)
	tmpl1 := template.Must(template.New("test1").Parse("Template 1"))
	tmpl2 := template.Must(template.New("test2").Parse("Template 2"))
	tmpl3 := template.Must(template.New("test3").Parse("Template 3"))

	cache.Put("key1", tmpl1)
	cache.Put("key2", tmpl2)

	if cache.Len() != 2 {
		t.Errorf("Expected cache size 2, got %d", cache.Len())
	}

	cache.Put("key3", tmpl3)

	if cache.Len() != 2 {
		t.Errorf("Expected cache size 2 after eviction, got %d", cache.Len())
	}

	_, found := cache.Get("key1")
	if found {
		t.Error("Expected key1 to be evicted")
	}

	_, found = cache.Get("key2")
	if !found {
		t.Error("Expected key2 to be in cache")
	}

	_, found = cache.Get("key3")
	if !found {
		t.Error("Expected key3 to be in cache")
	}
}

func TestTemplateCacheConcurrency(t *testing.T) {
	cache := NewTemplateCache(100)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := fmt.Sprintf("key%d", j)
				tmpl := template.Must(template.New(key).Parse(fmt.Sprintf("Template %d", j)))
				cache.Put(key, tmpl)
				cache.Get(key)
			}
		}(i)
	}

	wg.Wait()

	if cache.Len() > 100 {
		t.Errorf("Cache size %d exceeds max size 100", cache.Len())
	}
}

func TestHashTemplate(t *testing.T) {
	tmpl1 := "Hello {{.Name}}"
	tmpl2 := "Hello {{.Name}}"
	tmpl3 := "Goodbye {{.Name}}"

	hash1 := hashTemplate(tmpl1)
	hash2 := hashTemplate(tmpl2)
	hash3 := hashTemplate(tmpl3)

	if hash1 != hash2 {
		t.Error("Expected same hash for identical templates")
	}

	if hash1 == hash3 {
		t.Error("Expected different hashes for different templates")
	}

	if hashTemplate(tmpl1) != hash1 {
		t.Error("Expected consistent hash for same template")
	}
}
