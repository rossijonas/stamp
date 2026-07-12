package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

func TestHelloCmd(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"hello"}, []manager.Adapter{})
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "stamp — A lightweight yet powerful wrapper")
	assert.Contains(t, output, "stamp init")
	assert.Contains(t, output, "stamp doctor")
	assert.Contains(t, output, "stamp man install")
}

func TestRootCmd_DefaultSuggestsHello(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{}, []manager.Adapter{})
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Don't know where to start? Try:")
	assert.Contains(t, output, "stamp hello")
}
