package main

import (
	gitbom "github.com/git-bom/gitbom-go/pkg/cmd"
	"log"
)

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	if err := gitbom.Run(); err != nil {
		log.Fatalln(err)
	}
}
