package leveldbcache

import (
	"github.com/geziyor/geziyor/cache"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestDiskCache(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "cache")
	if err != nil {
		t.Fatalf("TempDir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	c, err := New(filepath.Join(tempDir, "Db"))
	if err != nil {
		t.Fatalf("New leveldb,: %v", err)
	}

	cache.PleaseCache(t, c)
}
