package main

import (
	"fmt"
	"github.com/facebookgo/symwalk"
	"github.com/fkautz/gitbom-go"
	"log"
	"os"
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
	err := symwalk.Walk(fileName, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer func(f *os.File) {
				err := f.Close()
				if err != nil {
					log.Printf("error closing %s: %s", path, err)
				}
			}(f)

			if err := gb.AddSha1ReferenceFromReader(f, nil, info.Size()); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}
