package geziyor

import (
	"io"
	"net/http"
)

// Request is a small wrapper around *http.Request that contains Metadata and Rendering option
type Request struct {
	*http.Request
	Meta      map[string]interface{}
	Rendered  bool
	Cancelled bool
}

// NewRequest returns a new Request given a method, URL, and optional body.
func NewRequest(method, url string, body io.Reader) (*Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	return &Request{Request: req}, nil
}
