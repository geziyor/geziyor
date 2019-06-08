package geziyor

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

var file *os.File
var once sync.Once

// Export exports response data as JSON streaming file
func Export(response *Response) {
	once.Do(func() {
		newFile, err := os.OpenFile("out.json", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "output file creation error: %v", err)
			return
		}
		file = newFile
	})

	for res := range response.Exports {
		//fmt.Println(res)
		_ = json.NewEncoder(file).Encode(res)
	}
}
