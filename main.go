package main

import (
	"encoding/xml"
	"errors"
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
			log.Printf("WARNING: TC Id (%s) is not in protocols map! Faulty processing of tc IDs or master xml!", id)
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

func GetTests(path string) (passed, failed []string) {
	passedMap := map[string]string{}
	failedMap := map[string]string{}

	files, err := GetFilesFromDir(path)
	if err != nil {
		log.Fatalf("Couldn't get files from dir (%s): %+v", path, err)
	}

	// Example filenames:
	// report_4AP2-65015_PASS_2022_07_08_14h_11m.xml
	// report_4AP2-38126_FAIL_2022_07_08_17h_53m.xml
	tcIdPattern, err := regexp.Compile("report_(?P<id>[a-zA-Z0-9]+-\\d+)_(?P<status>[A-Z]+)_.*")
	idIndex := tcIdPattern.SubexpIndex("id")
	statusIndex := tcIdPattern.SubexpIndex("status")

	for _, file := range files {
		filename := filepath.Base(file)
		if !tcIdPattern.MatchString(filename) {
			continue
		}

		tcId := tcIdPattern.FindStringSubmatch(filename)[idIndex]
		tcStatus := tcIdPattern.FindStringSubmatch(filename)[statusIndex]
		log.Println(tcId, tcStatus)

		if tcStatus == "PASS" {
			passedMap[tcId] = filename
		} else {
			failedMap[tcId] = filename
		}
	}

	for failedTc := range failedMap {
		// If failed TC is found in the passed TCs -> remove from failed
		// we don't care about a TCs intermediate status so long as it is passed in the end
		if _, found := passedMap[failedTc]; found {
			delete(failedMap, failedTc)
			continue
		}
		failed = append(failed, failedTc)
	}

	for passedTc := range passedMap {
		passed = append(passed, passedTc)
	}

	return passed, failed
}

func WriteXmlFile(path string, data *TaExport) error {
	out, err := xml.MarshalIndent(&data, " ", "\t")
	if err != nil {
		return err
	}

	return os.WriteFile(path, out, 0644)
}

func GetRemainingProtocols(passed, failed []*Protocol, protocolsMap ProtocolsMap) []*Protocol {
	remaining := []*Protocol{}
	seenMap := map[string]int{}

	for id := range protocolsMap {
		seenMap[id] = 0
	}

	for _, protocol := range passed {
		seenMap[protocol.Id] += 1
	}

	for _, protocol := range failed {
		seenMap[protocol.Id] += 1
	}

	for id, run_times := range seenMap {
		if run_times == 0 {
			protocol, ok := protocolsMap[id]
			if !ok {
				// NOTE: This should never happen since seenMap is created from protocolsMap
				log.Fatalf("Found ID(%s) in seenMap but not in protocolsMap! Error in processing of remaining TCs", id)
			}
			remaining = append(remaining, protocol)
		}
	}
	return remaining
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Running script in: \t%s\n\n", root)

	// TODO: remove remaining_tests.xml cause it will be regenerated

	// Make sure master file exists
	master, err := GetMasterFile(root)
	if err != nil {
		log.Fatal(err)
	}

	passedTestIds, failedTestIds := GetTests(root)

	log.Println("Passed", passedTestIds)
	log.Println("Failed", failedTestIds)

	masterData, err := os.ReadFile(master)
	if err != nil {
		log.Fatalf("Failed to read master file `%s`: %+v", master, err)
	}

	var taExport TaExport
	err = xml.Unmarshal(masterData, &taExport)
	if err != nil {
		log.Fatalf("Failed to unmarshal master file `%s`: %+v", master, err)
	}

	protocolsMap := GetProtocolsMap(&taExport)
	passedProtocols := GetProtocolsForIds(protocolsMap, passedTestIds)
	failedProtocols := GetProtocolsForIds(protocolsMap, failedTestIds)
	remainingProtocols := GetRemainingProtocols(passedProtocols, failedProtocols, protocolsMap)

	passedTaExport := taExport.Clone(passedProtocols)
	failedTaExport := taExport.Clone(failedProtocols)
	remainingTaExport := taExport.Clone(remainingProtocols)

	err = WriteXmlFile("passed.xml", passedTaExport)
	if err != nil {
		log.Fatalf("Failed to write `passed.xml`: %+v", err)
	}

	err = WriteXmlFile("failed.xml", failedTaExport)
	if err != nil {
		log.Fatalf("Failed to write `failed.xml`: %+v", err)
	}

	err = WriteXmlFile("remaining.xml", remainingTaExport)
	if err != nil {
		log.Fatalf("Failed to write `remaining.xml`: %+v", err)
	}
}
