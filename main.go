package main

import (
	"log"
	"os"
)

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	log.SetOutput(os.Stderr)

	StartHTTPServer()
}
