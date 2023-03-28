package middleware

import (
	"github.com/hohner2008/geziyor/client"
)

// RequestResponseProcessor interface is for middlewares that needs to process both requests and responses
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
