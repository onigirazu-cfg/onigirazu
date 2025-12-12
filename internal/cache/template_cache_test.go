package cache

import (
	"context"
	"testing"
	"text/template"
	"time"
)

func TestTemplateCache_GetSet(t *testing.T) {
	ctx := context.Background()
	cache := NewTemplateCache(5*time.Minute, 100)
	defer cache.Close()

	templateStr := "Hello {{ .name }}"
	tmpl, err := template.New("test").Parse(templateStr)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	// Set template in cache
	err = cache.Set(ctx, templateStr, tmpl)
	if err != nil {
		t.Fatalf("Failed to set template: %v", err)
	}

	// Get template from cache
	cachedTmpl, found := cache.Get(ctx, templateStr)
	if !found {
		t.Fatal("Template not found in cache")
	}

	if cachedTmpl == nil {
		t.Fatal("Cached template is nil")
	}

	// Verify it's the same template
	if cachedTmpl != tmpl {
		t.Error("Cached template is different from original")
	}
}

func TestTemplateCache_Miss(t *testing.T) {
	ctx := context.Background()
	cache := NewTemplateCache(5*time.Minute, 100)
	defer cache.Close()

	// Try to get non-existent template
	_, found := cache.Get(ctx, "non-existent template")
	if found {
		t.Error("Expected cache miss, got hit")
	}

	stats := cache.Stats()
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}
}

func TestTemplateCache_Expiration(t *testing.T) {
	ctx := context.Background()
	cache := NewTemplateCache(100*time.Millisecond, 100)
	defer cache.Close()

	templateStr := "Hello {{ .name }}"
	tmpl, err := template.New("test").Parse(templateStr)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	// Set template in cache
	err = cache.Set(ctx, templateStr, tmpl)
	if err != nil {
		t.Fatalf("Failed to set template: %v", err)
	}

	// Verify it's in cache
	_, found := cache.Get(ctx, templateStr)
	if !found {
		t.Fatal("Template not found in cache")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Try to get expired template
	_, found = cache.Get(ctx, templateStr)
	if found {
		t.Error("Expected expired template to be removed")
	}
}

func TestTemplateCache_GetOrParse(t *testing.T) {
	ctx := context.Background()
	cache := NewTemplateCache(5*time.Minute, 100)
	defer cache.Close()

	templateStr := "Hello {{ .name }}"
	funcMap := template.FuncMap{}

	// First call should parse and cache
	tmpl1, err := cache.GetOrParse(ctx, templateStr, funcMap)
	if err != nil {
		t.Fatalf("Failed to get or parse template: %v", err)
	}

	if tmpl1 == nil {
		t.Fatal("Template is nil")
	}

	stats := cache.Stats()
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	// Second call should hit cache
	tmpl2, err := cache.GetOrParse(ctx, templateStr, funcMap)
	if err != nil {
		t.Fatalf("Failed to get or parse template: %v", err)
	}

	if tmpl2 == nil {
		t.Fatal("Template is nil")
	}

	stats = cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}
}

func TestTemplateCache_Clear(t *testing.T) {
	ctx := context.Background()
	cache := NewTemplateCache(5*time.Minute, 100)
	defer cache.Close()

	// Add some templates
	for i := 0; i < 5; i++ {
		templateStr := "Template {{ .value }}"
		tmpl, _ := template.New("test").Parse(templateStr)
		_ = cache.Set(ctx, templateStr, tmpl)
	}

	if cache.Size() != 1 { // All templates are the same, so only 1 unique hash
		t.Errorf("Expected 1 entry, got %d", cache.Size())
	}

	// Clear cache
	err := cache.Clear(ctx)
	if err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}

	if cache.Size() != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", cache.Size())
	}
}

func TestTemplateCache_LRUEviction(t *testing.T) {
	ctx := context.Background()
	cache := NewTemplateCache(5*time.Minute, 3) // Max 3 entries
	defer cache.Close()

	// Add 4 templates (should evict the first one)
	templates := []string{
		"Template 1: {{ .a }}",
		"Template 2: {{ .b }}",
		"Template 3: {{ .c }}",
		"Template 4: {{ .d }}",
	}

	for _, templateStr := range templates {
		tmpl, _ := template.New("test").Parse(templateStr)
		_ = cache.Set(ctx, templateStr, tmpl)
	}

	// Cache should have 3 entries
	if cache.Size() != 3 {
		t.Errorf("Expected 3 entries, got %d", cache.Size())
	}

	// First template should be evicted
	_, found := cache.Get(ctx, templates[0])
	if found {
		t.Error("Expected first template to be evicted")
	}

	// Other templates should still be in cache
	for i := 1; i < 4; i++ {
		_, found := cache.Get(ctx, templates[i])
		if !found {
			t.Errorf("Expected template %d to be in cache", i)
		}
	}

	stats := cache.Stats()
	if stats.Evictions != 1 {
		t.Errorf("Expected 1 eviction, got %d", stats.Evictions)
	}
}

func TestTemplateCache_Stats(t *testing.T) {
	ctx := context.Background()
	cache := NewTemplateCache(5*time.Minute, 100)
	defer cache.Close()

	templateStr := "Hello {{ .name }}"
	tmpl, _ := template.New("test").Parse(templateStr)

	// Set template
	_ = cache.Set(ctx, templateStr, tmpl)

	// Hit
	cache.Get(ctx, templateStr)

	// Miss
	cache.Get(ctx, "non-existent")

	stats := cache.Stats()

	if stats.TotalEntries != 1 {
		t.Errorf("Expected 1 total entry, got %d", stats.TotalEntries)
	}

	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}

	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	expectedHitRate := 50.0 // 1 hit out of 2 total accesses
	if stats.HitRate != expectedHitRate {
		t.Errorf("Expected hit rate %.2f%%, got %.2f%%", expectedHitRate, stats.HitRate)
	}
}

func TestTemplateCache_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	cache := NewTemplateCache(5*time.Minute, 100)
	defer cache.Close()

	templateStr := "Hello {{ .name }}"
	funcMap := template.FuncMap{}

	// Concurrent reads and writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_, _ = cache.GetOrParse(ctx, templateStr, funcMap)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	stats := cache.Stats()
	if stats.TotalEntries != 1 {
		t.Errorf("Expected 1 entry, got %d", stats.TotalEntries)
	}

	// Should have 1 miss (first parse) and 999 hits
	totalAccesses := stats.Hits + stats.Misses
	if totalAccesses != 1000 {
		t.Errorf("Expected 1000 total accesses, got %d", totalAccesses)
	}
}

func TestTemplateCache_HashCollision(t *testing.T) {
	ctx := context.Background()
	cache := NewTemplateCache(5*time.Minute, 100)
	defer cache.Close()

	// Two different templates
	template1 := "Hello {{ .name }}"
	template2 := "Goodbye {{ .name }}"

	tmpl1, _ := template.New("test1").Parse(template1)
	tmpl2, _ := template.New("test2").Parse(template2)

	// Set both templates
	_ = cache.Set(ctx, template1, tmpl1)
	_ = cache.Set(ctx, template2, tmpl2)

	// Both should be retrievable
	cached1, found1 := cache.Get(ctx, template1)
	if !found1 {
		t.Error("Template 1 not found")
	}

	cached2, found2 := cache.Get(ctx, template2)
	if !found2 {
		t.Error("Template 2 not found")
	}

	// They should be different templates
	if cached1 == cached2 {
		t.Error("Expected different templates, got same")
	}

	if cache.Size() != 2 {
		t.Errorf("Expected 2 entries, got %d", cache.Size())
	}
}

func BenchmarkTemplateCache_GetOrParse(b *testing.B) {
	ctx := context.Background()
	cache := NewTemplateCache(5*time.Minute, 1000)
	defer cache.Close()

	templateStr := "Hello {{ .name }}, you are {{ .age }} years old"
	funcMap := template.FuncMap{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cache.GetOrParse(ctx, templateStr, funcMap)
		if err != nil {
			b.Fatalf("Failed to get or parse: %v", err)
		}
	}
}

func BenchmarkTemplateCache_DirectParse(b *testing.B) {
	templateStr := "Hello {{ .name }}, you are {{ .age }} years old"
	funcMap := template.FuncMap{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := template.New("test").Funcs(funcMap).Parse(templateStr)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
	}
}
