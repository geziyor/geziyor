package middleware

import (
	"github.com/geziyor/geziyor/client"
	"math/rand"
	"time"
)

// delay delays requests
type delay struct {
	requestDelayRandomize bool
	requestDelay          time.Duration
}

func NewDelay(requestDelayRandomize bool, requestDelay time.Duration) RequestProcessor {
	if requestDelayRandomize {
		rand.Seed(time.Now().UnixNano())
	}
	return &delay{requestDelayRandomize: requestDelayRandomize, requestDelay: requestDelay}
}

func (a *delay) ProcessRequest(r *client.Request) {
	if a.requestDelayRandomize {
		min := float64(a.requestDelay) * 0.5
		max := float64(a.requestDelay) * 1.5
		time.Sleep(a.requestDelay + time.Duration(rand.Intn(int(max-min))+int(min)))
	} else {
		time.Sleep(a.requestDelay)
	}
}
