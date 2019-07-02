package export

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestJSONExporter_Export(t *testing.T) {
	exporter := &JSON{
		FileName: "out.json",
		Indent:   " ",
	}
	_ = os.Remove(exporter.FileName)
	exports := make(chan interface{})
	go exporter.Export(exports)

	exports <- map[string]string{"key": "value"}
	close(exports)
	time.Sleep(time.Millisecond)

	contents, err := ioutil.ReadFile(exporter.FileName)
	assert.NoError(t, err)
	assert.Equal(t, "{\n \"key\": \"value\"\n}\n", string(contents))
}
