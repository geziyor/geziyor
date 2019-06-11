package exporter

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// JSONExporter exports response data as JSON streaming file
type JSONExporter struct {
	Filename   string
	EscapeHTML bool

	file *os.File
	once sync.Once
}

// Export exports response data as JSON streaming file
func (e JSONExporter) Export(exports chan interface{}) {

	// Default Filename
	if e.Filename == "" {
		e.Filename = "out.json"
	}

	// Create File
	e.once.Do(func() {
		newFile, err := os.OpenFile(e.Filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "output file creation error: %v", err)
			return
		}
		e.file = newFile
	})

	// Export data as responses came
	for res := range exports {
		encoder := json.NewEncoder(e.file)
		encoder.SetEscapeHTML(e.EscapeHTML)
		encoder.Encode(res)
	}
}
