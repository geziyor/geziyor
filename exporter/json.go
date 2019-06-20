package exporter

import (
	"encoding/json"
	"github.com/geziyor/geziyor/internal"
	"log"
	"os"
)

// JSONExporter exports response data as JSON streaming file
type JSONExporter struct {
	FileName   string
	EscapeHTML bool
}

// Export exports response data as JSON streaming file
func (e *JSONExporter) Export(exports chan interface{}) {

	// Create file
	newFile, err := os.OpenFile(internal.PreferFirst(e.FileName, "out.json"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Output file creation error: %v\n", err)
		return
	}

	encoder := json.NewEncoder(newFile)
	encoder.SetEscapeHTML(e.EscapeHTML)

	// Export data as responses came
	for res := range exports {
		if err := encoder.Encode(res); err != nil {
			log.Printf("JSON encoding error on exporter: %v\n", err)
		}
	}
}
