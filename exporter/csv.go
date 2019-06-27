package exporter

import (
	"encoding/csv"
	"fmt"
	"github.com/geziyor/geziyor/internal"
	"log"
	"os"
	"reflect"
	"sort"
)

// CSVExporter exports response data as CSV streaming file
type CSVExporter struct {
	FileName string
	Comma    rune
	UseCRLF  bool
}

// Export exports response data as CSV streaming file
func (e *CSVExporter) Export(exports chan interface{}) {

	// Create or append file
	file, err := os.OpenFile(internal.PreferFirst(e.FileName, "out.csv"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Output file creation error: %v\n", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Comma = internal.PreferFirstRune(e.Comma, ',')
	writer.UseCRLF = e.UseCRLF

	// Export data as responses came
	for res := range exports {
		var values []string

		// Detect type and extract CSV values
		val := reflect.ValueOf(res)
		switch val.Kind() {
		case reflect.Slice:
			for i := 0; i < val.Len(); i++ {
				values = append(values, fmt.Sprint(val.Index(i)))
			}
		case reflect.Map:
			iter := val.MapRange()
			for iter.Next() {
				values = append(values, fmt.Sprint(iter.Value()))
			}
			sort.Strings(values)
		}
		if err := writer.Write(values); err != nil {
			log.Printf("CSV writing error on exporter: %v\n", err)
		}
	}
	writer.Flush()
}
