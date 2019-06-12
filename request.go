package geziyor

import (
	"net/http"
)

// Request is a small wrapper around *http.Request that contains Metadata
type Request struct {
	*http.Request
	Meta map[string]interface{}
}
