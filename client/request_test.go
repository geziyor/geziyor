package client

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMeta(t *testing.T) {
	req, err := NewRequest("GET", "https://github.com/geziyor/geziyor", nil)
	assert.NoError(t, err)
	req.Meta["key"] = "value"

	assert.Equal(t, req.Meta["key"], "value")
}
