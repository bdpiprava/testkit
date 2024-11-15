package internal

import (
	"fmt"
	"log"
)

// PanicOnErr log error and panics if the error is not nil
func PanicOnErr(err error, msg string, args ...any) {
	if err != nil {
		log.Fatalf("%s: %v", fmt.Sprintf(msg, args...), err)
	}
}
