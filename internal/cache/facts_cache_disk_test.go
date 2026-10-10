package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFactsCacheOnDisk(t *testing.T) {
	dir := t.TempDir()
	fc := NewFactsCache(time.Hour)
	fc.UseDir(dir)
	fc.Set("web1", &SystemFacts{Hostname: "web1", OSFamily: "Debian"})
	if _, err := os.Stat(filepath.Join(dir, "web1.json")); err != nil {
		t.Fatalf("the entry is written: %v", err)
	}
	// a fresh cache (another run) reads it back
	again := NewFactsCache(time.Hour)
	again.UseDir(dir)
	facts, ok := again.Get("web1")
	if !ok || facts.OSFamily != "Debian" {
		t.Errorf("read back: %v %v", ok, facts)
	}
	// an expired file is ignored; Flush removes the files
	expired := NewFactsCache(time.Millisecond)
	expired.UseDir(dir)
	expired.Set("web2", &SystemFacts{Hostname: "web2"})
	time.Sleep(5 * time.Millisecond)
	if _, ok := NewFactsCacheDir(dir).Get("web2"); ok {
		t.Error("an expired entry on disk is a miss")
	}
	again.Flush()
	if files, _ := filepath.Glob(filepath.Join(dir, "*.json")); len(files) != 0 {
		t.Errorf("Flush removes the files: %v", files)
	}
}

func NewFactsCacheDir(dir string) *FactsCache {
	fc := NewFactsCache(time.Hour)
	fc.UseDir(dir)
	return fc
}
