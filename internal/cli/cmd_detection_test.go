package cli

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestLookPathFirst(t *testing.T) {
	tests := []struct {
		name      string
		available []string
		want      string
		wantOK    bool
	}{
		{name: "prefers the first candidate", available: []string{"apt", "apt-get"}, want: "apt", wantOK: true},
		{name: "falls back to the second", available: []string{"apt-get"}, want: "apt-get", wantOK: true},
		{name: "none available", available: nil, want: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookPathOrig := lookPath
			defer func() { lookPath = lookPathOrig }()
			lookPath = func(name string) (string, error) {
				for _, available := range tt.available {
					if name == available {
						return "/usr/bin/" + name, nil
					}
				}
				return "", fmt.Errorf("not found: %s", name)
			}

			got, ok := lookPathFirst("apt", "apt-get")
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDetectAdapters_FallbackToAptGet(t *testing.T) {
	lookPathOrig := lookPath
	defer func() { lookPath = lookPathOrig }()
	lookPath = func(name string) (string, error) {
		if name == "apt-get" {
			return "/usr/bin/apt-get", nil
		}
		return "", fmt.Errorf("not found: %s", name)
	}

	adapters := detectAdapters()
	count := 0
	for _, a := range adapters {
		if a.Name() == "apt" {
			count++
		}
	}
	assert.Equal(t, 1, count, "apt should be detected through apt-get when apt is absent")
}

func TestDetectAdapters_PrefersAptOverAptGet(t *testing.T) {
	lookPathOrig := lookPath
	defer func() { lookPath = lookPathOrig }()
	lookPath = func(name string) (string, error) {
		if name == "apt" || name == "apt-get" {
			return "/usr/bin/" + name, nil
		}
		return "", fmt.Errorf("not found: %s", name)
	}

	adapters := detectAdapters()
	count := 0
	for _, a := range adapters {
		if a.Name() == "apt" {
			count++
		}
	}
	assert.Equal(t, 1, count, "apt should register once when both apt and apt-get exist")
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

func TestDetectAdapters_YayWhenParuAbsent(t *testing.T) {
	lookPathOrig := lookPath
	defer func() { lookPath = lookPathOrig }()
	lookPath = func(name string) (string, error) {
		if name == "yay" {
			return "/usr/bin/yay", nil
		}
		return "", fmt.Errorf("not found: %s", name)
	}

	adapters := detectAdapters()
	require.Len(t, adapters, 1)
	assert.Equal(t, "yay", adapters[0].Name())
}

func TestDetectAdapters_PrefersParuOverYay(t *testing.T) {
	lookPathOrig := lookPath
	defer func() { lookPath = lookPathOrig }()
	lookPath = func(name string) (string, error) {
		if name == "paru" || name == "yay" {
			return "/usr/bin/" + name, nil
		}
		return "", fmt.Errorf("not found: %s", name)
	}

	adapters, warnings := detectAdaptersWithWarnings()
	require.Len(t, adapters, 1)
	assert.Equal(t, "paru", adapters[0].Name())
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "yay")
	assert.Contains(t, warnings[0], "using paru")
}
