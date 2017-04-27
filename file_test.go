package main

import (
	"testing"
)

func TestFiles(t *testing.T) {
	numFiles := 50

	f := fileOrder(numFiles)

	if len(f) != numFiles && len(f) != numFiles-1 {
		t.Errorf("Expected %d or %d, got %d", numFiles, numFiles-1, len(f))
	}

	count := make([]int, len(fileSizes))
	for _, file := range f {
		count[file.fileSizeIndex]++
		t.Logf("%+v\n", file)
	}

	if count[0] != 35 {
		t.Errorf("Expected 35 files in index 0, got %d", count[0])
	}
	if count[1] != 10 {
		t.Errorf("Expected 35 files in index 1, got %d", count[1])
	}
	if count[2] != 2 {
		t.Errorf("Expected 35 files in index 2, got %d", count[2])
	}
	if count[3] != 2 {
		t.Errorf("Expected 35 files in index 3, got %d", count[3])
	}

}
