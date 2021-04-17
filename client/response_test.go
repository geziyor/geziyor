package client

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestResponse_JoinURL(t *testing.T) {
	req, _ := NewRequest("GET", "https://localhost.com/test/a.html", nil)
	resp := Response{Request: req}

	joinedURL, err := resp.JoinURL("/source")
	assert.NoError(t, err)
	assert.Equal(t, "https://localhost.com/source", joinedURL.String())
}
