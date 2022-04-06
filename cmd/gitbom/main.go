package main

import (
	gitbom "github.com/git-bom/gitbom-go/pkg/cmd"
	"log"
)

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	gitbom.Cmd.Run()
}
