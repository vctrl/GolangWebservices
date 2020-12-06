package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
)

func main() {
	out := os.Stdout

	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	err := OSReadDir(out, path, "", printFiles)
	return err
}

func OSReadDir(out io.Writer, path string, prefix string, printFiles bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	fileInfo, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return err
	}

	if !printFiles {
		fileInfo = Filter(fileInfo, func(file os.FileInfo) bool {
			return file.IsDir()
		})
	}

	sort.Slice(fileInfo, func(i, j int) bool {
		return fileInfo[i].Name() < fileInfo[j].Name()
	})

	prefixSymbol := '├'
	currentPrefix := "│\t"
	for index, file := range fileInfo {
		if len(fileInfo) == 1 || index == len(fileInfo)-1 {
			prefixSymbol = '└'
			currentPrefix = "\t"
		}
		fileSize := ""
		if !file.IsDir() {
			size := file.Size()
			if size == 0 {
				fileSize = " (empty)"
			} else {
				fileSize = " (" + strconv.FormatInt(size, 10) + "b)" //todo use stringbuilder
			}
		}

		fmt.Fprintf(out, "%s%c───%s%s\n", prefix, prefixSymbol, file.Name(), fileSize)
		if file.IsDir() {
			OSReadDir(out, path+"/"+file.Name(), prefix+currentPrefix, printFiles)
		}
	}

	return nil
}

func Filter(vs []os.FileInfo, f func(os.FileInfo) bool) []os.FileInfo {
	vsf := make([]os.FileInfo, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}
