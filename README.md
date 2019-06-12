# Geziyor
Geziyor is a blazing fast web crawling and web scraping framework, used to crawl websites and extract structured data from their pages. It can be used for a wide range of purposes, from data mining to monitoring and automated testing.   

[![GoDoc](https://godoc.org/github.com/geziyor/geziyor?status.svg)](https://godoc.org/github.com/geziyor/geziyor)
[![report card](https://goreportcard.com/badge/github.com/geziyor/geziyor)](http://goreportcard.com/report/geziyor/geziyor)

## Features
- 1.000+ Requests/Sec
- Caching (Memory/Disk)
- Automatic Data Exporting (JSON, CSV, or custom)
- Limit Concurrency (Global/Per Domain)
- Request Delays (Constant/Randomized)
- Automatic response decoding to UTF-8

See scraper [Options](https://godoc.org/github.com/geziyor/geziyor#Options) for customization. 

## Usage
Simplest usage 

```go
geziyor.NewGeziyor(geziyor.Options{
    StartURLs: []string{"http://api.ipify.org"},
    ParseFunc: func(r *geziyor.Response) {
        fmt.Println(string(r.Body))
    },
}).Start()
```

## Status
We highly recommend you to use go modules. As this project is in **development stage** right now and **API is not stable**.


## Installation

    go get github.com/geziyor/geziyor
