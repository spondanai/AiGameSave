package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUpdateCacheRoundTrip(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "update_check.json")
	t.Setenv("AGS_UPDATE_CACHE", cachePath)

	want := updateCache{
		CheckedAt:     time.Now(),
		LatestVersion: "v0.1.0",
		LatestHash:    "abc123",
	}
	if err := writeUpdateCache(want); err != nil {
		t.Fatal(err)
	}

	got, ok := readUpdateCache()
	if !ok {
		t.Fatal("expected cache to be readable")
	}
	if got.LatestVersion != want.LatestVersion || got.LatestHash != want.LatestHash {
		t.Fatalf("unexpected cache: %#v", got)
	}
}

func TestLatestModuleInfoCachedUsesFreshCache(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "update_check.json")
	t.Setenv("AGS_UPDATE_CACHE", cachePath)

	if err := writeUpdateCache(updateCache{
		CheckedAt:     time.Now(),
		LatestVersion: "v9.9.9",
		LatestHash:    "cachedhash",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := latestModuleInfoCached()
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "v9.9.9" || got.Origin.Hash != "cachedhash" {
		t.Fatalf("expected cached module info, got %#v", got)
	}
}

func TestReadUpdateCacheIgnoresInvalidCache(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "update_check.json")
	t.Setenv("AGS_UPDATE_CACHE", cachePath)

	if err := os.WriteFile(cachePath, []byte("{"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, ok := readUpdateCache(); ok {
		t.Fatal("expected invalid cache to be ignored")
	}
}
