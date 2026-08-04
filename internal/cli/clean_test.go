package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

func TestCleanCmd_Scenarios(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"default run", []string{"clean", "-y"}, false},
		{"dry-run flag", []string{"clean", "--dry-run"}, false},
		{"manager flag", []string{"clean", "-m", "dnf", "-y"}, false},
		{"unknown manager", []string{"clean", "-m", "nonexistent"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := execCmd(t, tt.args, []manager.Adapter{&mockAdapter{name: "dnf"}})
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCleanCmd_WithCleanMock(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName: "brew",
			CleanResult: []string{"would remove 2 old versions"},
		},
	}
	buf, err := execCmd(t, []string{"clean", "--dry-run"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "would clean")
}
