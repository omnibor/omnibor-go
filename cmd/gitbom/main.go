package main

import (
	"fmt"
	"github.com/fkautz/gitbom-go"
	"log"
	"os"
	"path/filepath"
)

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	gb := gitbom.NewGitBom()
	for i := 1; i < len(os.Args); i++ {
		if err := addToGitBom(gb, os.Args[i]); err != nil {
			log.Fatalln(err)
		}
	}
	fmt.Println(gb.String())
}

func addToGitBom(gb gitbom.ArtifactTree, fileName string) error {
	err := filepath.Walk(fileName, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			if err := gb.AddSha1ReferenceFromReader(f, nil, info.Size()); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}
