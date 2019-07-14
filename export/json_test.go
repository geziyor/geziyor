package export

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestJSONLineExporter_Export(t *testing.T) {
	exporter := &JSONLine{
		FileName: "out.json",
		Indent:   " ",
	}
	_ = os.Remove(exporter.FileName)
	exports := make(chan interface{})
	go exporter.Export(exports)

	exports <- map[string]string{"key": "value"}
	close(exports)
	time.Sleep(time.Millisecond) // Wait for writing to disk

	contents, err := ioutil.ReadFile(exporter.FileName)
	assert.NoError(t, err)
	assert.Equal(t, "{\n \"key\": \"value\"\n}\n", string(contents))
}

func TestJSONExporter_Export(t *testing.T) {
	exporter := &JSON{
		FileName: "out.json",
	}
	_ = os.Remove(exporter.FileName)
	exports := make(chan interface{})
	go exporter.Export(exports)

	exports <- map[string]string{"key": "value"}
	close(exports)
	time.Sleep(time.Millisecond) // Wait for writing to disk

	contents, err := ioutil.ReadFile(exporter.FileName)
	assert.NoError(t, err)
	assert.Equal(t, "[\n\t{\"key\":\"value\"}\n]\n", string(contents))
}
