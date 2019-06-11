package exporter

import (
	"encoding/csv"
	"fmt"
	"github.com/geziyor/geziyor"
	"os"
	"reflect"
	"sync"
)

// CSVExporter exports response data as CSV streaming file
type CSVExporter struct {
	Filename string

	once   sync.Once
	file   *os.File
	writer *csv.Writer
}

func (e CSVExporter) Export(response *geziyor.Response) {

	// Default Filename
	if e.Filename == "" {
		e.Filename = "out.csv"
	}

	// Create File
	e.once.Do(func() {
		newFile, err := os.OpenFile(e.Filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "output file creation error: %v", err)
			return
		}
		e.file = newFile
		e.writer = csv.NewWriter(e.file)
	})

	// Export data as responses came
	for res := range response.Exports {
		var values []string

		val := reflect.ValueOf(res)
		switch val.Kind() {
		// TODO: Map type support is temporary. Ordering is wrong. Needs to be sorted by map keys (CSV headers).
		case reflect.Map:
			iter := val.MapRange()
			for iter.Next() {
				values = append(values, fmt.Sprint(iter.Value()))
			}

		case reflect.Slice:
			for i := 0; i < val.Len(); i++ {
				values = append(values, fmt.Sprint(val.Index(i)))
			}
		}

		e.writer.Write(values)
		e.writer.Flush()
	}
}
