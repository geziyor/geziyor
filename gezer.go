package gezer

import (
	"bytes"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

type Gezer struct {
	client *http.Client
	wg     sync.WaitGroup
	opt    Opt
}

type Opt struct {
	StartURLs []string
	ParseFunc func(response *Response)
}

type Response struct {
	*http.Response
	Body []byte
	Doc  *goquery.Document
}

func NewGezer(opt Opt) *Gezer {
	return &Gezer{
		client: &http.Client{
			Timeout: time.Second * 10,
		},
		opt: opt,
	}
}

func (g *Gezer) Start() {
	g.wg.Add(len(g.opt.StartURLs))

	for _, url := range g.opt.StartURLs {
		go g.getRequest(url)
	}

	g.wg.Wait()
}

func (g *Gezer) getRequest(url string) {
	defer g.wg.Done()

	// Get request
	resp, err := g.client.Get(url)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return
	}

	// Read body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	// Create Document
	doc, _ := goquery.NewDocumentFromReader(bytes.NewReader(body))

	// Create response
	response := Response{
		Response: resp,
		Body:     body,
		Doc:      doc,
	}

	// ParseFunc response
	g.opt.ParseFunc(&response)
}
