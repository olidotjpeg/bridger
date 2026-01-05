package indexer

import (
	"context"
	"testing"
)

func TestIsSupported(test *testing.T) {
	// Arrange
	testPath := "helloWorld.png"

	// Act
	testResult := IsSupported(testPath)

	// Assert
	if !testResult {
		test.Error("This file is not supported")
	}

	if testResult {
		test.Log("Path is supported")
	}
}

func TestIsSupportedWithBadPath(test *testing.T) {
	// Arrange
	testPath := "helloWorld.mp4"

	// Act
	testResult := IsSupported(testPath)

	// Assert
	if !testResult {
		test.Log("This file is not supported, this is right for mp4")
	}

	if testResult {
		test.Error("This extension is somehow supported, not right for mp4")
	}
}

func TestRead(t *testing.T) {
	// Arrange
	reader := &ExifReader{}

	ctx, _ := context.WithCancel(context.Background())
	testPath := "helloWorld.png"

	// Act
	result, err := reader.Read(ctx, testPath)

	// Assert
	if result != nil {
		t.Log(result)
	} else {
		t.Error(err)
	}
}
