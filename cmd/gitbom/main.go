package main

import (
	_ "github.com/fkautz/gitbom-go/pkg/cmd"
	"github.com/rwxrob/cmdbox"
	"log"
)

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	cmdbox.Execute()
}
