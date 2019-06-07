package gezer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"net/http"
	"os"
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

func (g *Gezer) Get(url string) {
	g.wg.Add(1)
	go g.getRequest(url)
}

func (g *Gezer) getRequest(url string) {
	defer g.wg.Done()

	// Log
	fmt.Println("Fetching: ", url)

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
		Gezer:    g,
		Exports:  make(chan map[string]interface{}, 1),
	}

	// Export Function
	go func() {
		file, err := os.Create("out.json")
		if err != nil {
			fmt.Fprintf(os.Stderr, "output file creation error: %v", err)
			return
		}

		for res := range response.Exports {
			fmt.Println(res)
			_ = json.NewEncoder(file).Encode(res)
		}

	}()

	// ParseFunc response
	g.opt.ParseFunc(&response)

}
