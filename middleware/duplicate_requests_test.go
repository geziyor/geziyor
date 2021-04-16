package middleware

import (
	"github.com/geziyor/geziyor/client"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestDuplicateRequests_ProcessRequest(t *testing.T) {
	longURL := "https://example.com" + strings.Repeat("/path", 50)
	req, err := client.NewRequest("GET", longURL, nil)
	assert.NoError(t, err)
	req2, err := client.NewRequest("GET", longURL, nil)
	assert.NoError(t, err)

	duplicateRequestsProcessor := DuplicateRequests{RevisitEnabled: false}
	duplicateRequestsProcessor.ProcessRequest(req)
	duplicateRequestsProcessor.ProcessRequest(req2)
	duplicateRequestsProcessor.ProcessRequest(req2)
	duplicateRequestsProcessor.ProcessRequest(req2)
}
