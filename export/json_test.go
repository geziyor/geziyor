package export

import "testing"

func TestJSONExporter_Export(t *testing.T) {
	ch := make(chan interface{})
	defer close(ch)

	exporter := &JSON{
		FileName: "out.json",
		Indent:   " ",
	}
	go exporter.Export(ch)

	ch <- map[string]string{"key": "value"}
}
