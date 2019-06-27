package exporter

import "testing"

func TestCSVExporter_Export(t *testing.T) {
	ch := make(chan interface{})
	defer close(ch)

	exporter := &CSVExporter{
		FileName: "test.out",
		Comma:    ';',
	}
	go exporter.Export(ch)

	ch <- []string{"1", "2"}
}
