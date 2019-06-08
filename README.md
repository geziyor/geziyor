# Gezer
Scraper and crawler framework for Golang. Gezer uses go *channels* over *callbacks*   

## Features
- 1.000+ Requests/Sec  
- Caching
- Automatic Data Exporting


## Example
```go
gezer := NewGezer(Opt{
    StartURLs: []string{"http://quotes.toscrape.com/"},
    ParseFunc: func(r *Response) {
        r.Doc.Find("div.quote").Each(func(i int, s *goquery.Selection) {
            // Export Data
            r.Exports <- map[string]interface{}{
                "text":   s.Find("span.text").Text(),
                "author": s.Find("small.author").Text(),
                "tags": s.Find("div.tags > a.tag").Map(func(_ int, s *goquery.Selection) string {
                    return s.Text()
                }),
            }
        })

        // Next Page
        if href, ok := r.Doc.Find("li.next > a").Attr("href"); ok {
            go r.Gezer.Get(r.JoinURL(href))
        }
    },
})
gezer.Start()
```


## Installation

    go get github.com/gogezer/gezer