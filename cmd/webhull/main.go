package main

import (
	"log"

	"github.com/layer87-labs/webhull/cmd/webhull/cmd"
)

func main() {
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
