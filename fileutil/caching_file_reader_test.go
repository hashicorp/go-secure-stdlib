package fileutil

import (
	"os"
	"testing"
	"time"
)

func TestCachingFileReader(t *testing.T) {
	content1 := "before"
	content2 := "after"

	// Create temporary file.
	f, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Error(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	r := NewCachingFileReader(f.Name(), 1*time.Minute)
	currentTime := time.Now()
	r.setStaticTime(currentTime)

	// Write initial content to file and check that we can read it.
	os.WriteFile(f.Name(), []byte(content1), 0o644)
	got, err := r.ReadFile()
	if err != nil {
		t.Error(err)
	}
	if got != content1 {
		t.Errorf("got '%s', expected '%s'", got, content1)
	}

	// Write new content to the file.
	os.WriteFile(f.Name(), []byte(content2), 0o644)

	// Advance simulated time, but not enough for cache to expire.
	currentTime = currentTime.Add(30 * time.Second)
	r.setStaticTime(currentTime)

	// Read again and check we still got the old cached content.
	got, err = r.ReadFile()
	if err != nil {
		t.Error(err)
	}
	if got != content1 {
		t.Errorf("got '%s', expected '%s'", got, content1)
	}

	// Advance simulated time for cache to expire.
	currentTime = currentTime.Add(30 * time.Second)
	r.setStaticTime(currentTime)

	// Read again and check that we got the new content.
	got, err = r.ReadFile()
	if err != nil {
		t.Error(err)
	}
	if got != content2 {
		t.Errorf("got '%s', expected '%s'", got, content2)
	}
}
