package middleware

import (
	"github.com/geziyor/geziyor/client"
)

type RequestResponseProcessor interface {
	RequestProcessor
	ResponseProcessor
}

// RequestProcessor called before requests made.
// Set request.Cancelled = true to cancel request
type RequestProcessor interface {
	ProcessRequest(r *client.Request)
}

// ResponseProcessor called after request response receive
type ResponseProcessor interface {
	ProcessResponse(r *client.Response)
}
