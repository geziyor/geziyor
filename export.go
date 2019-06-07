package gezer

import (
	"encoding/json"
	"fmt"
	"os"
)

func Export(response *Response) {
	file, err := os.Create("out.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "output file creation error: %v", err)
		return
	}

	for res := range response.Exports {
		//fmt.Println(res)
		_ = json.NewEncoder(file).Encode(res)
	}
}
