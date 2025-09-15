package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRandomContent(t *testing.T) {
	original, err := NewRandomContent()
	require.NoError(t, err)

	parsed, err := ContentFrom(original.Raw)
	require.NoError(t, err)

	assert.Equal(t, original.CID.String(), parsed.CID.String())
}
