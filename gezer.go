package gezer

import (
	"io/ioutil"
	"net/http"
	"time"
)

type Gezer struct {
	client  *http.Client
	Results chan *Response
}

type Response struct {
	*http.Response
	Body []byte
}

func NewGezer() *Gezer {
	return &Gezer{
		client: &http.Client{
			Timeout: time.Second * 10,
		},
		Results: make(chan *Response, 1),
	}
}

func (g *Gezer) StartURLs(urls ...string) {
	for _, url := range urls {

		// Get request
		resp, err := g.client.Get(url)
		if err != nil {
			if resp != nil {
				_ = resp.Body.Close()
			}
			continue
		}

		// Read body
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			continue
		}
		_ = resp.Body.Close()

		// Create response
		response := Response{
			Response: resp,
			Body:     body,
		}

		// Send response
		g.Results <- &response
	}

	// Close chan, as we finished sending all the results
	close(g.Results)
}
