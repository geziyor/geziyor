package export

import (
	"encoding/json"
	"github.com/geziyor/geziyor/internal"
	"log"
	"os"
)

// JSON exports response data as JSON streaming file
type JSON struct {
	FileName   string
	EscapeHTML bool
	Prefix     string
	Indent     string
}

// Export exports response data as JSON streaming file
func (e *JSON) Export(exports chan interface{}) {

	// Create or append file
	file, err := os.OpenFile(internal.DefaultString(e.FileName, "out.json"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Output file creation error: %v\n", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(e.EscapeHTML)
	encoder.SetIndent(e.Prefix, e.Indent)

	// Export data as responses came
	for res := range exports {
		if err := encoder.Encode(res); err != nil {
			log.Printf("JSON encoding error on exporter: %v\n", err)
		}
	}
}
