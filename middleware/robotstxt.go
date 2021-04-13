package middleware

import (
	"github.com/geziyor/geziyor/client"
	"github.com/geziyor/geziyor/internal"
	"github.com/geziyor/geziyor/metrics"
	"github.com/temoto/robotstxt"
	"strconv"
	"sync"
)

// RobotsTxt middleware filters out requests forbidden by the robots.txt exclusion standard.
type RobotsTxt struct {
	metrics        *metrics.Metrics
	robotsDisabled bool
	client         *client.Client
	mut            sync.RWMutex
	robotsMap      map[string]*robotstxt.RobotsData
}

func NewRobotsTxt(client *client.Client, metrics *metrics.Metrics, robotsDisabled bool) RequestProcessor {
	return &RobotsTxt{
		metrics:        metrics,
		robotsDisabled: robotsDisabled,
		client:         client,
		robotsMap:      make(map[string]*robotstxt.RobotsData),
	}
}

func (m *RobotsTxt) ProcessRequest(r *client.Request) {
	if m.robotsDisabled {
		return
	}

	// TODO: Locking like this improves performance but sometimes it causes duplicate requests to robots.txt
	m.mut.RLock()
	robotsData, exists := m.robotsMap[r.Host]
	m.mut.RUnlock()

	if !exists {
		robotsReq, err := client.NewRequest("GET", r.URL.Scheme+"://"+r.Host+"/robots.txt", nil)
		if err != nil {
			return // Don't Do anything
		}

		m.metrics.RobotsTxtRequestCounter.Add(1)
		robotsResp, err := m.client.DoRequest(robotsReq)
		if err != nil {
			return // Don't Do anything
		}
		m.metrics.RobotsTxtResponseCounter.With("status", strconv.Itoa(robotsResp.StatusCode)).Add(1)

		robotsData, err = robotstxt.FromStatusAndBytes(robotsResp.StatusCode, robotsResp.Body)
		if err != nil {
			return // Don't Do anything
		}

		m.mut.Lock()
		m.robotsMap[r.Host] = robotsData
		m.mut.Unlock()
	}

	if !robotsData.TestAgent(r.URL.Path, r.UserAgent()) {
		m.metrics.RobotsTxtForbiddenCounter.With("method", r.Method).Add(1)
		internal.Logger.Println("Forbidden by robots.txt:", r.URL.String())
		r.Cancel()
	}
}
