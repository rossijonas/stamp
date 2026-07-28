package cli

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectAdapters_Runs(t *testing.T) {
	t.Parallel()
	adapters := detectAdapters()
	assert.NotNil(t, adapters)
}

func TestDetectAdapters_ParuPreferredOverPacman(t *testing.T) {
	lookPathOrig := lookPath
	defer func() { lookPath = lookPathOrig }()
	lookPath = func(name string) (string, error) {
		if name == "paru" {
			return "/usr/bin/paru", nil
		}
		return "", fmt.Errorf("not found: %s", name)
	}

	adapters := detectAdapters()
	hasParu := false
	hasPacman := false
	for _, a := range adapters {
		if a.Name() == "paru" {
			hasParu = true
		}
		if a.Name() == "pacman" {
			hasPacman = true
		}
	}
	assert.True(t, hasParu, "paru should be detected when paru binary is present")
	assert.False(t, hasPacman, "pacman should NOT be detected when paru is present")
}

func TestDetectAdapters_FallbackToPacman(t *testing.T) {
	lookPathOrig := lookPath
	defer func() { lookPath = lookPathOrig }()
	lookPath = func(name string) (string, error) {
		if name == "pacman" {
			return "/usr/bin/pacman", nil
		}
		return "", fmt.Errorf("not found: %s", name)
	}

	adapters := detectAdapters()
	hasParu := false
	hasPacman := false
	for _, a := range adapters {
		if a.Name() == "paru" {
			hasParu = true
		}
		if a.Name() == "pacman" {
			hasPacman = true
		}
	}
	assert.False(t, hasParu, "paru should NOT be detected when only pacman binary is present")
	assert.True(t, hasPacman, "pacman should be detected when pacman binary is present")
}

func TestDetectAdapters_NpmDetected(t *testing.T) {
	lookPathOrig := lookPath
	defer func() { lookPath = lookPathOrig }()
	lookPath = func(name string) (string, error) {
		if name == "npm" {
			return "/usr/bin/npm", nil
		}
		return "", fmt.Errorf("not found: %s", name)
	}

	adapters := detectAdapters()
	hasNpm := false
	for _, a := range adapters {
		if a.Name() == "npm" {
			hasNpm = true
			break
		}
	}
	assert.True(t, hasNpm, "npm should be detected when npm binary is present")
}

func TestDetectAdapters_CargoDetected(t *testing.T) {
	lookPathOrig := lookPath
	defer func() { lookPath = lookPathOrig }()
	lookPath = func(name string) (string, error) {
		if name == "cargo" {
			return "/usr/bin/cargo", nil
		}
		return "", fmt.Errorf("not found: %s", name)
	}

	adapters := detectAdapters()
	hasCargo := false
	for _, a := range adapters {
		if a.Name() == "cargo" {
			hasCargo = true
			break
		}
	}
	assert.True(t, hasCargo, "cargo should be detected when cargo binary is present")
}

func TestDetectAdapters_NeitherParuNorPacman(t *testing.T) {
	lookPathOrig := lookPath
	defer func() { lookPath = lookPathOrig }()
	lookPath = func(name string) (string, error) {
		return "", fmt.Errorf("not found: %s", name)
	}

	adapters := detectAdapters()
	for _, a := range adapters {
		if a.Name() == "paru" || a.Name() == "pacman" {
			t.Fatalf("neither paru nor pacman should be detected: got %s", a.Name())
		}
	}
}
