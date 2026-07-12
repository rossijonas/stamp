package main

import (
	"os"
	"testing"
)

func TestMainFunc(_ *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"stamp", "--help"}
	main()
}
