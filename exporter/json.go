package exporter

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
)

// JSONExporter exports response data as JSON streaming file
type JSONExporter struct {
	FileName   string
	EscapeHTML bool

	once    sync.Once
	mut     sync.Mutex
	encoder *json.Encoder
}

// Export exports response data as JSON streaming file
func (e *JSONExporter) Export(exports chan interface{}) {

	// Default filename
	if e.FileName == "" {
		e.FileName = "out.json"
	}

	// Create file
	e.once.Do(func() {
		newFile, err := os.OpenFile(e.FileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "output file creation error: %v", err)
			return
		}
		e.encoder = json.NewEncoder(newFile)
		e.encoder.SetEscapeHTML(e.EscapeHTML)
	})

	// Export data as responses came
	for res := range exports {
		if err := e.encoder.Encode(res); err != nil {
			log.Printf("JSON encoding error on exporter: %v\n", err)
		}
	}
}
