package export

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCSVExporter_Export(t *testing.T) {
	exporter := &CSV{
		FileName: "out.csv",
		Comma:    ';',
	}
	_ = os.Remove(exporter.FileName)
	exports := make(chan interface{})
	go exporter.Export(exports)

	exports <- []string{"1", "2"}
	exports <- map[string]string{"key1": "value1", "key2": "value2"}
	close(exports)
	time.Sleep(time.Millisecond)

	contents, err := ioutil.ReadFile(exporter.FileName)
	assert.NoError(t, err)
	assert.Equal(t, "1;2\nvalue1;value2\n", string(contents))
}
