# Geziyor
Geziyor is a blazing fast web crawling and web scraping framework. It can be used to crawl websites and extract structured data from them. Geziyor is useful for a wide range of purposes such as data mining, monitoring and automated testing. 

[![GoDoc](https://godoc.org/github.com/geziyor/geziyor?status.svg)](https://godoc.org/github.com/geziyor/geziyor)
[![report card](https://goreportcard.com/badge/github.com/geziyor/geziyor)](http://goreportcard.com/report/geziyor/geziyor)

## Features
- 1.000+ Requests/Sec
- JS Rendering
- Caching (Memory/Disk)
- Automatic Data Exporting (JSON, CSV, or custom)
- Metrics (Prometheus, Expvar, or custom)
- Limit Concurrency (Global/Per Domain)
- Request Delays (Constant/Randomized)
- Cookies and Middlewares
- Automatic response decoding to UTF-8

See scraper [Options](https://godoc.org/github.com/geziyor/geziyor#Options) for all custom settings. 

## Status
The project is in **development phase**. Thus, we highly recommend you to use Geziyor with go modules.

## Examples
Simple usage 

```go
geziyor.NewGeziyor(&geziyor.Options{
    StartURLs: []string{"http://api.ipify.org"},
    ParseFunc: func(g *geziyor.Geziyor, r *geziyor.Response) {
        fmt.Println(string(r.Body))
    },
}).Start()
```

Advanced usage

```go
func main() {
    geziyor.NewGeziyor(&geziyor.Options{
        StartURLs: []string{"http://quotes.toscrape.com/"},
        ParseFunc: quotesParse,
        Exporters: []geziyor.Exporter{exporter.JSON{}},
    }).Start()
}

func quotesParse(g *geziyor.Geziyor, r *geziyor.Response) {
    r.HTMLDoc.Find("div.quote").Each(func(i int, s *goquery.Selection) {
        g.Exports <- map[string]interface{}{
            "text":   s.Find("span.text").Text(),
            "author": s.Find("small.author").Text(),
        }
    })
    if href, ok := r.HTMLDoc.Find("li.next > a").Attr("href"); ok {
        g.Get(r.JoinURL(href), quotesParse)
    }
}
```

See [tests](https://github.com/geziyor/geziyor/blob/master/geziyor_test.go) for more usage examples.

## Documentation

### Installation

    go get github.com/geziyor/geziyor

**NOTE**: macOS limits the maximum number of open file descriptors.
If you want to make concurrent requests over 256, you need to increase limits.
Read [this](https://wilsonmar.github.io/maximum-limits/) for more.

### Making Requests

Initial requests start with ```StartURLs []string``` field in ```Options```. 
Geziyor makes concurrent requests to those URLs.
After reading response, ```ParseFunc func(g *Geziyor, r *Response)``` called.

```go
geziyor.NewGeziyor(&geziyor.Options{
    StartURLs: []string{"http://api.ipify.org"},
    ParseFunc: func(g *geziyor.Geziyor, r *geziyor.Response) {
        fmt.Println(string(r.Body))
    },
}).Start()
```

If you want to manually create first requests, set ```StartRequestsFunc```.
```StartURLs``` won't be used if you create requests manually.  
You can make requests using ```Geziyor``` [methods](https://godoc.org/github.com/geziyor/geziyor#Geziyor):

```go
geziyor.NewGeziyor(&geziyor.Options{
    StartRequestsFunc: func(g *geziyor.Geziyor) {
    	g.Get("https://httpbin.org/anything", g.Opt.ParseFunc)
        g.GetRendered("https://httpbin.org/anything", g.Opt.ParseFunc)
        g.Head("https://httpbin.org/anything", g.Opt.ParseFunc)
    },
    ParseFunc: func(g *geziyor.Geziyor, r *geziyor.Response) {
        fmt.Println(string(r.Body))
    },
}).Start()
``` 

### Extracting Data

#### Extractors
You can add [Extractor]() to []Extractors option to extract structured data. 
Exporters need to be defined in order to extractors work. 

```go
geziyor.NewGeziyor(&geziyor.Options{
    StartURLs: []string{"https://www.theverge.com/2019/6/27/18760384/facebook-libra-currency-cryptocurrency-money-transfer-bank-problems-india-china"},
    Extractors: []geziyor.Extractor{
        &extractor.Text{Name: "title", Selector: ".c-page-title"},
        &extractor.Text{Name: "byline", Selector: ".c-byline__item:nth-child(1) > a"},
        &extractor.Text{Name: "summary", Selector: ".c-entry-summary"},
        &extractor.Text{Name: "content", Selector: ".c-entry-content"},
    },
    Exporters: []geziyor.Exporter{&exporter.JSON{}},
}).Start()
```    

#### HTML selectors

We can extract HTML elements using ```response.HTMLDoc```. HTMLDoc is Goquery's [Document](https://godoc.org/github.com/PuerkitoBio/goquery#Document).

HTMLDoc can be accessible on Response if response is HTML and can be parsed using Go's built-in HTML [parser](https://godoc.org/golang.org/x/net/html#Parse)
If response isn't HTML, ```response.HTMLDoc``` would be ```nil```.  

```go
geziyor.NewGeziyor(&geziyor.Options{
    StartURLs: []string{"http://quotes.toscrape.com/"},
    ParseFunc: func(g *geziyor.Geziyor, r *geziyor.Response) {
        r.HTMLDoc.Find("div.quote").Each(func(_ int, s *goquery.Selection) {
            log.Println(s.Find("span.text").Text(), s.Find("small.author").Text())
        })
    },
}).Start()
```

### Exporting Data

You can export data automatically using exporters. Just send data to ```Geziyor.Exports``` chan.
[Available exporters](https://godoc.org/github.com/geziyor/geziyor/exporter)

```go
geziyor.NewGeziyor(&geziyor.Options{
    StartURLs: []string{"http://quotes.toscrape.com/"},
    ParseFunc: func(g *geziyor.Geziyor, r *geziyor.Response) {
        r.HTMLDoc.Find("div.quote").Each(func(_ int, s *goquery.Selection) {
            g.Exports <- map[string]interface{}{
                "text":   s.Find("span.text").Text(),
                "author": s.Find("small.author").Text(),
            }
        })
    },
    Exporters: []geziyor.Exporter{&exporter.JSON{}},
}).Start()
```

## Roadmap

If you're interested in helping this project, please consider these features:

- Command line tool for: pausing and resuming scraper etc. (like [this](https://docs.scrapy.org/en/latest/topics/commands.html))
- Automatic item extractors (like [this](https://github.com/andrew-d/goscrape#goscrape))
- Deploying Scrapers to Cloud
- ~~Automatically exporting extracted data to multiple places (AWS, FTP, DB, JSON, CSV etc)~~ 
- Downloading media (Images, Videos etc) (like [this](https://docs.scrapy.org/en/latest/topics/media-pipeline.html))
- ~~Realtime metrics (Prometheus etc.)~~

  