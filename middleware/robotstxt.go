package middleware

import (
	"github.com/geziyor/geziyor/client"
	"github.com/temoto/robotstxt"
	"log"
	"sync"
)

// RobotsTxt middleware filters out requests forbidden by the robots.txt exclusion standard.
type RobotsTxt struct {
	robotsDisabled bool
	client         *client.Client
	mut            sync.RWMutex
	robotsMap      map[string]*robotstxt.RobotsData
}

func NewRobotsTxt(client *client.Client, robotsDisabled bool) RequestProcessor {
	return &RobotsTxt{
		robotsDisabled: robotsDisabled,
		client:         client,
		robotsMap:      make(map[string]*robotstxt.RobotsData),
	}
}

func (m *RobotsTxt) ProcessRequest(r *client.Request) {
	if m.robotsDisabled {
		return
	}

	// TODO: Locking like this improves performance but causes duplicate requests to robots.txt,
	m.mut.RLock()
	robotsData, exists := m.robotsMap[r.Host]
	m.mut.RUnlock()

	if !exists {
		// TODO: Disable retry
		robotsReq, err := client.NewRequest("GET", r.URL.Scheme+"://"+r.Host+"/robots.txt", nil)
		if err != nil {
			return // Don't Do anything
		}

		robotsResp, err := m.client.DoRequestClient(robotsReq)
		if err != nil {
			return // Don't Do anything
		}

		robotsData, err = robotstxt.FromStatusAndBytes(robotsResp.StatusCode, robotsResp.Body)
		if err != nil {
			return // Don't Do anything
		}

		m.mut.Lock()
		m.robotsMap[r.Host] = robotsData
		m.mut.Unlock()
	}

	if !robotsData.TestAgent(r.URL.Path, r.UserAgent()) {
		// TODO: Forbidden requests metrics
		log.Println("Forbidden by robots.txt:", r.URL.String())
		r.Cancel()
	}
}
