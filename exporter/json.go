package exporter

import (
	"encoding/json"
	"fmt"
	"github.com/geziyor/geziyor"
	"log"
	"os"
	"sync"
)

// JSONExporter exports response data as JSON streaming file
type JSONExporter struct {
	FileName   string
	EscapeHTML bool

	once sync.Once
	file *os.File
}

// Export exports response data as JSON streaming file
func (e JSONExporter) Export(response *geziyor.Response) {

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
		e.file = newFile
	})

	// Export data as responses came
	for res := range response.Exports {
		encoder := json.NewEncoder(e.file)
		encoder.SetEscapeHTML(e.EscapeHTML)
		if err := encoder.Encode(res); err != nil {
			log.Printf("JSON encoding error on exporter: %v\n", err)
		}
	}
}
