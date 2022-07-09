package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
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

func CheckMasterExists(path string) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		fmt.Println(file.Name(), file.IsDir())
		// TODO: check master in file.Name()
	}
}

func GetTests(path string) []string {
	testIds := []string{}
	files, err := GetFilesFromDir(path)
	if err != nil {
		// TODO: this should be handled better
		panic(err)
	}

	tcIdPattern, err := regexp.Compile("report_(?P<id>[a-zA-Z0-9]+-\\d+)_.*")
	idIndex := tcIdPattern.SubexpIndex("id")

	for _, file := range files {
		filename := filepath.Base(file)
		// TODO: what if not submatch is found, does the indexing fail ?
		tcId := tcIdPattern.FindStringSubmatch(filename)[idIndex]
		testIds = append(testIds, tcId)
	}
	return testIds
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

	// TODO: no need to check for folders, we can just extract all xml reports
	//       from the current dir and take their status from the filename
	//       If an id is found as passed and failed we only take the passed status.
	//       The final output should be passed.xml, failed.xml, remaining.xml
	//       The reason for havin failed.xml generated is so that its easier to rerun tests
	// Make sure expected paths are created
	CheckDirExists(passedPath)
	CheckDirExists(failedPath)
	CheckMasterExists(root)

	passedTestIds := GetTests(passedPath)
	failedTestIds := GetTests(failedPath)

	log.Println("Passed", passedTestIds)
	log.Println("Failed", failedTestIds)
}
