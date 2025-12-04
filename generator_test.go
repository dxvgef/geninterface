package main

import (
	"testing"
)

func Test_generator(t *testing.T) {
	err := generator(
		"pkg/test_1.go",
		[]string{"RuntimeConfig", "Config"},
		true,
		0644,
		true,
		"Accessor",
		".accessor",
		".getter",
		".setter",
	)
	if err != nil {
		t.Fatal(err)
	}
}
