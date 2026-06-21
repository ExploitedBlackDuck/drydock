package composeparse_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/adapters/composeparse"
	"github.com/drydock/drydock/internal/core/compose"
)

func service(stack compose.DesiredStack, name string) (compose.DesiredService, bool) {
	for _, s := range stack.Services {
		if s.Name == name {
			return s, true
		}
	}
	return compose.DesiredService{}, false
}

func TestParseBuildsDesiredStack(t *testing.T) {
	stack, err := composeparse.Parse(context.Background(), "app", "testdata", []string{"testdata/compose.yaml"})
	require.NoError(t, err)

	require.Len(t, stack.Services, 2)

	web, ok := service(stack, "web")
	require.True(t, ok)
	assert.Equal(t, "nginx:1.27", web.Image)
	assert.False(t, web.HasAnonymousVolumes)

	db, ok := service(stack, "db")
	require.True(t, ok)
	assert.Equal(t, "postgres:16", db.Image)
	assert.True(t, db.HasAnonymousVolumes, "the bare /var/lib/scratch mount is an anonymous volume")

	assert.Contains(t, stack.Volumes, "dbdata")
	assert.Contains(t, stack.Networks, "backend")
}

func TestParseMissingFileErrors(t *testing.T) {
	_, err := composeparse.Parse(context.Background(), "app", "testdata", []string{"testdata/nope.yaml"})
	assert.Error(t, err)
}
