package main

import (
	"log"

	"github.com/zeroqual/playerzzz/internal/ui"
)

func main() {
	if err := ui.Run(); err != nil {
		log.Fatal(err)
	}
}
