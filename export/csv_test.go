package export

import "testing"

func TestCSVExporter_Export(t *testing.T) {
	ch := make(chan interface{})
	defer close(ch)

	exporter := &CSV{
		FileName: "out.csv",
		Comma:    ';',
	}
	go exporter.Export(ch)

	ch <- []string{"1", "2"}
	ch <- map[string]string{"key1": "value1", "key2": "value2"}
}
