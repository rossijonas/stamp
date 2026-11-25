package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYay_Identity(t *testing.T) {
	t.Parallel()
	m := NewYay()
	assert.Equal(t, "yay", m.Name())
	assert.Equal(t, "yay", m.Binary())
}

func TestYay_InstallArgs(t *testing.T) {
	t.Parallel()
	m := NewYay()
	var gotName string
	var gotArgs []string
	m.exec = func(_ context.Context, name string, args ...string) ([]byte, error) {
		gotName = name
		gotArgs = args
		return nil, nil
	}

	require.NoError(t, m.Install(WithYes(context.Background()), "htop"))
	assert.Equal(t, "yay", gotName, "no sudo wrapper")
	assert.Equal(t, []string{
		"-S", "--noconfirm",
		"--answerclean", "None",
		"--answerdiff", "None",
		"--answeredit", "None",
		"htop",
	}, gotArgs)
}

func TestYay_UpdateFullArgs(t *testing.T) {
	t.Parallel()
	m := NewYay()
	var gotArgs []string
	m.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		gotArgs = args
		return nil, nil
	}

	require.NoError(t, m.Update(WithYes(context.Background()), ""))
	assert.Equal(t, []string{
		"-Syu", "--noconfirm",
		"--answerclean", "None",
		"--answerdiff", "None",
		"--answeredit", "None",
		"--answerupgrade", "None",
	}, gotArgs)
}

func TestYay_RemoveArgs(t *testing.T) {
	t.Parallel()
	m := NewYay()
	var gotArgs []string
	m.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		gotArgs = args
		return nil, nil
	}

	require.NoError(t, m.Remove(WithYes(context.Background()), "htop"))
	assert.Equal(t, []string{"-Rs", "--noconfirm", "htop"}, gotArgs, "remove has no build menus")
}

func TestYay_CleanArgs(t *testing.T) {
	t.Parallel()
	m := NewYay()
	var gotName string
	var gotArgs []string
	m.exec = func(_ context.Context, name string, args ...string) ([]byte, error) {
		gotName = name
		gotArgs = args
		return nil, nil
	}

	_, err := m.Clean(WithYes(context.Background()), false)
	require.NoError(t, err)
	assert.Equal(t, "yay", gotName, "user cache, no sudo")
	assert.Equal(t, []string{"-Sc", "--noconfirm"}, gotArgs)
}

func TestYay_RefreshArgs(t *testing.T) {
	t.Parallel()
	m := NewYay()
	var gotName string
	m.exec = func(_ context.Context, name string, _ ...string) ([]byte, error) {
		gotName = name
		return nil, nil
	}

	require.NoError(t, m.Refresh(context.Background()))
	assert.Equal(t, "yay", gotName)
}

func TestYay_SearchInfoCheck(t *testing.T) {
	t.Parallel()
	m := NewYay()
	m.exec = mockExecutorHelper("aur/htop 3.2.2-1\n", nil)

	res, err := m.Search(WithYes(context.Background()), "htop")
	require.NoError(t, err)
	assert.Equal(t, []string{"htop"}, res)

	m.exec = mockExecutorHelper("Name: htop\n", nil)
	info, err := m.Info(WithYes(context.Background()), "htop")
	require.NoError(t, err)
	assert.Contains(t, info, "htop")

	m.exec = mockExecutorHelper("htop 3.2.1 -> 3.2.2\n", nil)
	updates, err := m.CheckUpdate(WithYes(context.Background()), "")
	require.NoError(t, err)
	require.Len(t, updates, 1)
	assert.Equal(t, "htop", updates[0].Package)
}

func TestYay_ListInstalled(t *testing.T) {
	t.Parallel()
	m := NewYay()
	m.exec = mockExecutorHelper("htop\nripgrep\n", nil)
	pkgs, err := m.ListInstalled(context.Background())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"htop", "ripgrep"}, pkgs)
}

func TestYay_RefusesRoot(t *testing.T) {
	old := geteuid
	geteuid = func() int { return 0 }
	t.Cleanup(func() { geteuid = old })

	m := NewYay()
	m.exec = mockExecutorHelper("", nil)

	err := m.Install(WithYes(context.Background()), "htop")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must run as your user")
}

func TestYay_SpecGrammar(t *testing.T) {
	t.Parallel()
	// yay follows pacman grammar: repo-qualified refs are accepted.
	_, err := ParseSpecForManager("yay", "extra/htop")
	require.NoError(t, err)
	_, err = ParseSpecForManager("yay", "htop=3.2.1")
	require.NoError(t, err)
}
