package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

type Protocol struct {
	XMLName             xml.Name `xml:"protocol"`
	ProjectId           string   `xml:"project-id,attr"`
	Id                  string   `xml:"id,attr"`
	TestScriptReference string   `xml:"test-script-reference"`
}

type DVPlan struct {
	XMLName          xml.Name    `xml:"dv-plan"`
	ProjectId        string      `xml:"project-id,attr"`
	Id               string      `xml:"id,attr"`
	BuildResult      string      `xml:"build-result"`
	VerificationLoop string      `xml:"verification-loop"`
	Protocols        []*Protocol `xml:"protocols>protocol"`
}

type TaExport struct {
	XMLName xml.Name `xml:"ta-tool-export"`
	DvPlan  DVPlan   `xml:"dv-plan"`
}

func (t *TaExport) Clone(protocols []*Protocol) *TaExport {
	return &TaExport{
		DvPlan: DVPlan{
			ProjectId:        t.DvPlan.ProjectId,
			Id:               t.DvPlan.Id,
			BuildResult:      t.DvPlan.BuildResult,
			VerificationLoop: t.DvPlan.VerificationLoop,
			Protocols:        protocols,
		},
	}
}

type ProtocolsMap map[string]*Protocol

func GetProtocolsMap(taExport *TaExport) ProtocolsMap {
	out := ProtocolsMap{}

	for _, protocol := range taExport.DvPlan.Protocols {
		out[protocol.Id] = protocol
	}
	return out
}

func GetProtocolsForIds(protocolsMap ProtocolsMap, tcIds []string) []*Protocol {
	out := []*Protocol{}

	for _, id := range tcIds {
		protocol, ok := protocolsMap[id]
		if !ok {
			log.Fatalf("TC Id (%s) is not in protocols map! Faulty processing of tc IDs or master xml!", id)
		}
		out = append(out, protocol)
	}

	return out
}

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

func GetMasterFile(path string) (string, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return "", err
	}

	masterPattern, err := regexp.Compile("^master_.*?\\.xml$")
	if err != nil {
		return "", err
	}

	for _, file := range files {
		fileName := file.Name()
		if masterPattern.MatchString(fileName) {
			log.Printf("Found master file '%s'\n", fileName)
			return fileName, nil
		}
	}
	return "", errors.New("Coudln't find master `master_*.xml` file. Please make sure its in the current directory")
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

	// Make sure master file exists
	master, err := GetMasterFile(root)
	if err != nil {
		panic(err)
	}
	log.Println(master)

	passedTestIds := GetTests(passedPath)
	failedTestIds := GetTests(failedPath)

	log.Println("Passed", passedTestIds)
	log.Println("Failed", failedTestIds)

	masterData, err := os.ReadFile(master)
	if err != nil {
		panic(err)
	}

	var taExport TaExport
	err = xml.Unmarshal(masterData, &taExport)
	if err != nil {
		panic(err)
	}

	/*
			fmt.Println(taExport.DvPlan.ProjectId, taExport.DvPlan.Id)
			fmt.Println(taExport.DvPlan.BuildResult, taExport.DvPlan.VerificationLoop)
			fmt.Println(
		        taExport.DvPlan.Protocols[0].Id,
		        taExport.DvPlan.Protocols[0].ProjectId,
		        taExport.DvPlan.Protocols[0].TestScriptReference
		    )
	*/
	protocolsMap := GetProtocolsMap(&taExport)
	passedProtocols := GetProtocolsForIds(protocolsMap, passedTestIds)
	failedProtocols := GetProtocolsForIds(protocolsMap, failedTestIds)

	passedTaExport := taExport.Clone(passedProtocols)
	failedTaExport := taExport.Clone(failedProtocols)
	fmt.Println(passedTaExport, failedTaExport)

	// TODO: 1. Marshal and write passed & failed export to file
	// TODO: 2. Compute the remaining TCs and marshal & write to file
}
