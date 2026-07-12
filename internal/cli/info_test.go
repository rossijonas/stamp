package cli

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

func TestInfoCmd_MultiManager(t *testing.T) {
	t.Parallel()
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName: "dnf",
			InfoResult:  "Name: htop\nVersion: 3.4.1\nSummary: process viewer",
		},
		&manager.Mock{
			ManagerName: "brew",
			InfoErr:     errors.New("not available"),
		},
	}

	buf, err := execCmd(t, []string{"info", "htop"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "htop:")
	assert.Contains(t, output, "dnf:")
	assert.Contains(t, output, "v3.4.1")
	assert.Contains(t, output, "brew:")
	assert.Contains(t, output, "not available")
}

func TestInfoCmd_RawManagerBlock(t *testing.T) {
	t.Parallel()
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName: "dnf",
			InfoResult:  "Name: htop\nVersion: 3.4.1\nSummary: process viewer\nDescription: interactive process viewer",
		},
	}

	buf, err := execCmd(t, []string{"info", "htop", "-m", "dnf"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "htop via dnf:")
	assert.Contains(t, output, "Description: interactive process viewer")
}

func TestInfoCmd_JSON(t *testing.T) {
	t.Parallel()
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName: "dnf",
			InfoResult:  "Name: htop\nVersion: 3.4.1",
		},
	}

	buf, err := execCmd(t, []string{"info", "htop", "--json"}, adapters)
	require.NoError(t, err)

	var report infoReport
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	assert.Equal(t, "htop", report.Package)
	require.Len(t, report.Results, 1)
	assert.Equal(t, "dnf", report.Results[0].Manager)
	assert.True(t, report.Results[0].Found)
	assert.Contains(t, report.Results[0].Info, "Version: 3.4.1")
}

func TestInfoCmd_NotFound(t *testing.T) {
	t.Parallel()
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName: "dnf",
			InfoErr:     errors.New("not found"),
		},
	}

	buf, err := execCmd(t, []string{"info", "htop"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "htop: not found in any package manager")
}

func TestInfoCmd_Errors(t *testing.T) {
	t.Parallel()
	// Invalid name
	_, err := execCmd(t, []string{"info", "-invalid"}, []manager.Adapter{})
	require.Error(t, err)

	// Unknown manager
	_, err = execCmd(t, []string{"info", "htop", "-m", "nonexistent"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.Error(t, err)
}
