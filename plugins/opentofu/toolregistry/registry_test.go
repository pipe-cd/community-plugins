package toolregistry

import (
	"context"
	"os/exec"
	"testing"

	"github.com/pipe-cd/piped-plugin-sdk-go/toolregistry/toolregistrytest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_OpenTofu(t *testing.T) {
	t.Parallel()

	c := toolregistrytest.NewTestToolRegistry(t)

	r := NewRegistry(c)

	p, err := r.OpenTofu(context.Background(), "1.9.1")
	require.NoError(t, err)
	require.NotEmpty(t, p)

	out, err := exec.CommandContext(context.Background(), p, "version").CombinedOutput()
	require.NoError(t, err)

	expected := "OpenTofu v1.9.1"

	assert.Contains(t, string(out), expected)
}
