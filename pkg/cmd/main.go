package cmd

import (
	"bytes"
	"fmt"
	"github.com/facebookgo/symwalk"
	"github.com/fkautz/gitbom-go"
	"github.com/rwxrob/bonzai"
	"github.com/rwxrob/cmdbox/util"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sync"
)

var Cmd = &bonzai.Cmd{
	Name:      `gitbom`,
	Summary:   `gitbom`,
	Usage:     `[gitbom]`,
	Version:   `v0.0.1`,
	Copyright: `Copyright 2021 gitbom-go contributors`,
	License:   `Apache-2`,
	Commands:  []*bonzai.Cmd{helpCmd, artifactTreeCmd, bomCmd},

	Description: `
		The foo commands do foo stuff. You can start the description here
		and wrap it to look nice and it will just work. Otherwise, just
		follow the same guidelines as for Go documentation. Note that the
		x.Call Method here is omitted since the main work is delegated to
		the subcommands in the command tree. The help command, however, is
		the default because it is first. `,

	// no Call since has Commands, if had Call would only call if
	// commands didn't match
	Call: func(caller *bonzai.Cmd, args ...string) error {
		printHelp()
		return nil
	},
}

var artifactTreeCmd = &bonzai.Cmd{
	Name: "artifact-tree",
	Call: artifactTreeCall,
}

var bomCmd = &bonzai.Cmd{
	Name: "bom",
	Call: bomCall,
}

var helpCmd = &bonzai.Cmd{
	Name: "help",
	Call: helpCall,
}

func helpCall(caller *bonzai.Cmd, args ...string) error {
	printHelp()
	return nil
}

func artifactTreeCall(caller *bonzai.Cmd, args ...string) error {
	wg := startAgents()
	if len(args) == 0 {
		printHelp()
		return nil
	}

	gb := gitbom.NewGitBom()
	for i := 0; i < len(args); i++ {
		if err := addPathToGitbom(gb, args[i], agentChan); err != nil {
			log.Println(args[i], err)
			return err
		}
	}

	close(agentChan)
	wg.Wait()

	// generate target gitbom with artifact tree
	if err := writeObject(".bom", gb); err != nil {
		log.Println(err)
		return err
	}

	fmt.Println(gb.Identity())

	return nil
}

var agentChan chan fileEvent = make(chan fileEvent)

func startAgents() *sync.WaitGroup {
	agentCount := 0
	wg := &sync.WaitGroup{}
	if runtime.GOMAXPROCS(0) < runtime.NumCPU() {
		agentCount = runtime.GOMAXPROCS(0)
	} else {
		agentCount = runtime.NumCPU()
	}
	for i := 0; i < agentCount; i++ {
		wg.Add(1)
		go agent(agentChan, wg, i)
	}
	return wg
}

func bomCall(caller *bonzai.Cmd, args ...string) error {
	if len(args) == 0 {
		printHelp()
		return nil
	}

	wg := sync.WaitGroup{}

	gb := gitbom.NewGitBom()

	// generate artifact tree
	for i := 2; i < len(args); i++ {
		if err := addPathToGitbom(gb, args[i], agentChan); err != nil {
			return err
		}
	}

	close(agentChan)
	wg.Wait()

	// generate target gitbom with artifact tree
	if err := writeObject(".bom", gb); err != nil {
		return err
	}

	gb2 := gitbom.NewGitBom()
	info, err := os.Stat(args[0])
	if err != nil {
		return err
	}
	if err = addFileToGitbom(args[0], info, gb2, gb); err != nil {
		return err
	}

	if err := writeObject(".bom", gb2); err != nil {
		return err
	}

	fmt.Println(gb2.Identity())
	return nil
}

func writeObject(prefix string, gb gitbom.ArtifactTree) error {
	objectDir := path.Join(prefix, "object", gb.Identity()[:2])
	objectPath := path.Join(objectDir, gb.Identity()[2:])
	if err := os.MkdirAll(objectDir, 0755); err != nil {
		log.Println(err)
		return err
	}
	if err := ioutil.WriteFile(objectPath, []byte(gb.String()), 0644); err != nil {
		return err
	}
	return nil
}

func addPathToGitbom(gb gitbom.ArtifactTree, fileName string, agentChan chan<- fileEvent) error {
	err := symwalk.Walk(fileName, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		path, err = filepath.EvalSymlinks(path)
		if err != nil {
			log.Println("ERROR", err)
			return err
		}
		info, err = os.Stat(path)
		if err != nil {
			log.Println("ERROR", err)
			return err
		}
		if !info.IsDir() {
			e := fileEvent{
				path: path,
				info: info,
				gb:   gb,
			}
			agentChan <- e
			return nil
		}
		return nil
	})
	return err
}

type fileEvent struct {
	path string
	info os.FileInfo
	gb   gitbom.ArtifactTree
}

func agent(e <-chan fileEvent, wg *sync.WaitGroup, id int) {
	defer wg.Done()
	for ev := range e {
		err2 := addFileToGitbom(ev.path, ev.info, ev.gb, nil)
		if err2 != nil {
			log.Println("ERROR", ev.path)
		}
	}
}

func addFileToGitbom(path string, info os.FileInfo, gb gitbom.ArtifactTree, identifier gitbom.Identifier) error {
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

	reader2 := &bytes.Buffer{}
	reader1 := io.TeeReader(f, reader2)
	if err := gb.AddSha1ReferenceFromReader(reader1, identifier, info.Size()); err != nil {
		return err
	}
	if err := gb.AddSha256ReferenceFromReader(reader2, identifier, info.Size()); err != nil {
		return err
	}
	return nil
}

func printHelp() (int, error) {
	return fmt.Println(util.Emph("**NAME**", 0, -1) + `
       gitbom (v0.0.1) - Generate gitboms from files

` + util.Emph("**USAGE**", 0, 01) + `
       gitbom artifact-tree [files]
       gitbom bom [artifact-file] [artifact-tree-files [artifact-tree files...]]

       gitbom will create a .bom/ directory in the current working
       directory and store generated gitboms in .bom/

` + util.Emph("**LEGAL**", 0, 01) + `
       gitbom (v0.0.1) Copyright 2022 gitbom-go contributors
       SPDX-License-Identifier: Apache-2.0
`)
}
