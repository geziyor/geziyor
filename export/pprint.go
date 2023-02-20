package export

import (
	"encoding/json"
	"fmt"
)

// PrettyPrint logs exported data to console as pretty printed
type PrettyPrint struct{}

// Export logs exported data to console as pretty printed
func (*PrettyPrint) Export(exports chan interface{}) error {
	for res := range exports {
		dat, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			continue
		}
		fmt.Println(string(dat))
	}
	return nil
}
