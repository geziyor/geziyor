package gezer

import (
	"fmt"
	"testing"
)

func TestGezer_StartURLs(t *testing.T) {
	gezer := NewGezer()
	gezer.StartURLs("https://api.ipify.org")
	for result := range gezer.Results {
		fmt.Println(string(result.Body))
	}
}
