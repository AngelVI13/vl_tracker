package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func GetFilesFromDir(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func CheckDirExists(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Fatalf("'%s' folder does not exist. Please create it!", filepath.Base(path))
	}
}

func GetPassedTests(path string) []string {
	passedTestIds := []string{}
	files, err := GetFilesFromDir(path)
	if err != nil {
		panic(err)
	}
	fmt.Println(files)
	return passedTestIds
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Running script in: \t%s\n\n", root)

	// TODO: remove remaining_tests.xml cause it will be regenerated

	passedPath := filepath.Join(root, "passed")
	failedPath := filepath.Join(root, "failed")

	// Make sure expected paths are created
	CheckDirExists(passedPath)
	CheckDirExists(failedPath)

	passedTestIds := GetPassedTests(passedPath)
	log.Println(passedTestIds)
}
