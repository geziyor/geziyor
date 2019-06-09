# Geziyor
Geziyor is a blazing fast web crawling and web scraping framework, used to crawl websites and extract structured data from their pages. It can be used for a wide range of purposes, from data mining to monitoring and automated testing.   

[![GoDoc](https://godoc.org/github.com/geziyor/geziyor?status.svg)](https://godoc.org/github.com/geziyor/geziyor)
[![report card](https://goreportcard.com/badge/github.com/geziyor/geziyor)](http://goreportcard.com/report/geziyor/geziyor)

## Features
- 1.000+ Requests/Sec
- Caching
- Automatic Data Exporting
- Limit Concurrency Global/Per Domain
- Automatic response decoding to UTF-8


## Usage
Simplest usage 

```go
geziyor.NewGeziyor(geziyor.Options{
    StartURLs: []string{"http://api.ipify.org"},
    ParseFunc: func(r *geziyor.Response) {
    	fmt.Println(r.Doc.Text())
    },
}).Start()
```

Export all quotes and authors to out.json file.

```go
geziyor := NewGeziyor(Opt{
    StartURLs: []string{"http://quotes.toscrape.com/"},
    ParseFunc: func(r *Response) {
        r.Doc.Find("div.quote").Each(func(i int, s *goquery.Selection) {
            // Export Data
            r.Exports <- map[string]interface{}{
                "text":   s.Find("span.text").Text(),
                "author": s.Find("small.author").Text(),
            }
        })

        // Next Page
        if href, ok := r.Doc.Find("li.next > a").Attr("href"); ok {
            go r.Geziyor.Get(r.JoinURL(href))
        }
    },
})
geziyor.Start()
```


## Installation

    go get github.com/geziyor/geziyor
    
We highly recommend you to use go modules. As this project is in **development stage** right now.
