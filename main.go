package main

import (
	"embed"
	"log"
	"os"
)

//go:embed all:frontend/dist
var frontendDist embed.FS

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	log.SetOutput(os.Stderr)

	StartHTTPServer()
}
