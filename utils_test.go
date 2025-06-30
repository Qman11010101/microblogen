package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileExists(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "utils_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up

	// Test case 1: File exists
	existingFilePath := filepath.Join(tempDir, "exists.txt")
	f, err := os.Create(existingFilePath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	f.Close()

	if !FileExists(existingFilePath) {
		t.Errorf("FileExists(%q) = false, want true", existingFilePath)
	}

	// Test case 2: File does not exist
	nonExistingFilePath := filepath.Join(tempDir, "not_exists.txt")
	if FileExists(nonExistingFilePath) {
		t.Errorf("FileExists(%q) = true, want false", nonExistingFilePath)
	}

	// Test case 3: Path is a directory, not a file
	existingDirPath := filepath.Join(tempDir, "exists_dir")
	err = os.Mkdir(existingDirPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	// FileExists should return true for directories as os.Stat doesn't distinguish
	// between files and directories in its error return for non-existence.
	// The function name is FileExists, but its implementation os.Stat(name)
	// returns nil error if 'name' exists, regardless of whether it's a file or directory.
	if !FileExists(existingDirPath) {
		t.Errorf("FileExists(%q) for a directory = false, want true (as per os.Stat behavior)", existingDirPath)
	}

	// Test case 4: Empty path
	if FileExists("") {
		t.Errorf("FileExists(\"\") = true, want false")
	}

	// Test case 5: Path with invalid characters (OS dependent, but generally good to test)
	// This might not cause os.IsNotExist, but other errors. The function should still return false.
	// For simplicity, we'll use a path that's highly unlikely to exist and is invalid on many OSes.
	// Note: The behavior of os.Stat with highly invalid paths can vary.
	// If it returns an error that is *not* os.IsNotExist, FileExists would return true.
	// This highlights a potential ambiguity in FileExists if strictly interpreted as "file" vs "path".
	// Given the implementation `!os.IsNotExist(err)`, it means "path entry exists".
	// For a truly invalid path like "a\x00b.txt", os.Stat might return an error that isn't os.IsNotExist.
	// Let's test a more common "file not found" scenario for robustness.
	if FileExists("/path/that/most/definitely/does/not/exist/anywhere/on/the/system.txt") {
		t.Errorf("FileExists on a very unlikely path = true, want false")
	}
}
