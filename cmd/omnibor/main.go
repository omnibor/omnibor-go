package main

import (
	omnibor "github.com/omnibor/omnibor-go/pkg/cmd"
	"log"
)

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	if err := omnibor.Run(); err != nil {
		log.Fatalln(err)
	}
}
