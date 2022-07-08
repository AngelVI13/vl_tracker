package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func FilePathWalkDir(root string) ([]string, error) {
    var files []string
    err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if !info.IsDir() {
            files = append(files, path)
            fmt.Println("File ", info.Name())
        } else {
        	fmt.Println("Dir ", info.Name())
        }
        return nil
    })
    return files, err
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Running script in: \t%s\n\n", root)
	
	files, err := FilePathWalkDir(root)
    if err != nil {
        panic(err)
    }
    fmt.Println(files[0])
}
