package main

import (
	"log"

	"github.com/dmitriko/wtctrl/cmd"
)

func main() {
	root := cmd.RootCmd()
	if err := root.Execute(); err != nil {
		log.Fatal(err)
	}
}
