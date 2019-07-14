package export

import (
	"bytes"
	"encoding/json"
	"github.com/geziyor/geziyor/internal"
	"log"
	"os"
)

// JSONLine exports response data as JSON streaming file
type JSONLine struct {
	FileName   string
	EscapeHTML bool
	Prefix     string
	Indent     string
}

// Export exports response data as JSON streaming file
func (e *JSONLine) Export(exports chan interface{}) {

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

// JSON exports response data as JSON
type JSON struct {
	FileName   string
	EscapeHTML bool
}

// Export exports response data as JSON
func (e *JSON) Export(exports chan interface{}) {

	// Create or append file
	file, err := os.OpenFile(internal.DefaultString(e.FileName, "out.json"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Output file creation error: %v\n", err)
		return
	}
	defer file.Close()

	file.Write([]byte("[\n"))

	// Export data as responses came
	for res := range exports {
		data, err := jsonMarshalLine(res, e.EscapeHTML)
		if err != nil {
			log.Printf("JSON encoding error on exporter: %v\n", err)
			continue
		}
		file.Write(data)
	}

	// Override on last comma
	stat, err := file.Stat()
	if err != nil {
		file.Write([]byte("]\n"))
		return
	}
	file.WriteAt([]byte("\n]\n"), stat.Size()-2)
}

// jsonMarshalLine adds tab and comma around actual data
func jsonMarshalLine(t interface{}, escapeHTML bool) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(escapeHTML)

	buffer.Write([]byte("	"))         // Tab char
	err := encoder.Encode(t)          // Write actual data
	buffer.Truncate(buffer.Len() - 1) // Remove last newline char
	buffer.Write([]byte(",\n"))       // Write comma and newline char

	return buffer.Bytes(), err
}
