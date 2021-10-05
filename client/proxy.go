package client

import (
	"context"
	"github.com/geziyor/geziyor/internal"
	"net/http"
	"net/url"
	"sync/atomic"
)

const ProxyURLKey = 0

type roundRobinProxy struct {
	proxyURLs []*url.URL
	index     uint32
}

func (r *roundRobinProxy) GetProxy(pr *http.Request) (*url.URL, error) {
	index := atomic.AddUint32(&r.index, 1) - 1
	u := r.proxyURLs[index%uint32(len(r.proxyURLs))]

	// Set proxy url to context
	ctx := context.WithValue(pr.Context(), ProxyURLKey, u.String())
	*pr = *pr.WithContext(ctx)
	return u, nil
}

// RoundRobinProxy creates a proxy switcher function which rotates
// ProxyURLs on every request.
// The proxy type is determined by the URL scheme. "http", "https"
// and "socks5" are supported. If the scheme is empty,
// "http" is assumed.
func RoundRobinProxy(proxyURLs ...string) func(*http.Request) (*url.URL, error) {
	if len(proxyURLs) < 1 {
		return http.ProxyFromEnvironment
	}
	parsedProxyURLs := make([]*url.URL, len(proxyURLs))
	for i, u := range proxyURLs {
		parsedURL, err := url.Parse(u)
		if err != nil {
			internal.Logger.Printf("proxy url parse: %v", err)
			return nil
		}
		parsedProxyURLs[i] = parsedURL
	}
	return (&roundRobinProxy{parsedProxyURLs, 0}).GetProxy
}
