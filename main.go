package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type ResultStruct struct {
	File string
	Line int
}

func searchTextInFile(path, text string) ([]int, error) {
	var result []int
	f, err := os.Open(path)
	if err != nil {
		return result, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	line := 1
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), text) {
			result = append(result, line)
		}
		line++
	}
	if err := scanner.Err(); err != nil {
		return result, err
	}
	return result, nil
}

func readPath(path string, filesChan chan string) {
	err := filepath.Walk(path, func(f string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			filesChan <- f
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}

func searchInFiles(out chan ResultStruct, c <-chan string, str string) {
	opened := true
	file := ""
	for opened {
		file, opened = <-c
		lines, err := searchTextInFile(file, str)
		if err == nil {
			for _, n := range lines {
				if n > 0 {
					out <- ResultStruct{file, n}
				}
			}
		}
	}
	close(out)
}


func main() {
	var wgSearchers, wgPrinter sync.WaitGroup

	flag.Parse()
	flag.Usage = func() {
		fmt.Printf("usage : %s searchText path1 path2 ... pathN \n", os.Args[0])
		os.Exit(0)
	}

	if flag.NArg() < 2 {
		flag.Usage()
		os.Exit(-1)
	}

	search := ""
	filesChan := make(chan string)
	for i, arg := range flag.Args() {
		if i == 0 {
			search = arg
		} else {
			wgSearchers.Add(1)
			go func(path string) {
				defer wgSearchers.Done()
				fmt.Printf("Collecting files in path %s ...\n", path)
				readPath(path, filesChan)
			}(arg)
		}
	}
	resultChan := make(chan ResultStruct)
	go searchInFiles(resultChan, filesChan, search)

	wgPrinter.Add(1)
	go func() {
		defer wgPrinter.Done()
		ok := true
		var i ResultStruct
		for ok {
			i, ok = <-resultChan
			if !ok {
				break
			}
			fmt.Printf("\t* Found %s in %s at line %d\n", search, i.File, i.Line)
		}
	}()

	wgSearchers.Wait()
	close(filesChan)
	wgPrinter.Wait()
}
