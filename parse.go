package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"regexp"
)

type moduleVersion struct {
	Module  string
	Current string
	Latest  string
}

type packageJSON map[string]interface{}

func main() {
	versions := npmOutdated(".")
	fmt.Println(versions)
	readPackageJSON(versions[0].Module, versions[0].Latest)
}

func npmOutdated(path string) []moduleVersion {
	cmd := exec.Command("npm", "outdated")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Dir = path
	err := cmd.Run()

	if err != nil {
		log.Fatal(err)
	}

	reader := bytes.NewReader(out.Bytes())
	scanner := bufio.NewScanner(reader)

	npmRegex, _ := regexp.Compile(`(\w+)\s+(\d+\.\d+\.\d+)\s+(\d+\.\d+\.\d+)\s+(\d+\.\d+\.\d+)`)

	lineIndex := 0

	version := make([]moduleVersion, 0, 0)
	for scanner.Scan() {
		if lineIndex != 0 {
			result := npmRegex.FindStringSubmatch(scanner.Text())
			version = append(version, moduleVersion{
				Module:  result[1],
				Current: result[2],
				Latest:  result[4],
			})
		}
		lineIndex++
	}

	return version
}

func readPackageJSON(module string, version string) {
	packageJSONBuffer, _ := ioutil.ReadFile("./package.json")
	var parsedPackageJSON packageJSON

	json.Unmarshal(packageJSONBuffer, &parsedPackageJSON)

	updateJSON(&parsedPackageJSON, "dependencies", module, version)
	updateJSON(&parsedPackageJSON, "devDependencies", module, version)
	updateJSON(&parsedPackageJSON, "optionalDependencies", module, version)

	updatedJSON, _ := json.MarshalIndent(&parsedPackageJSON, "", "  ")
	ioutil.WriteFile("./package.json", updatedJSON, 0770)
}

func updateJSON(parsedPackageJSON *packageJSON, key string, module string, version string) {
	_, ok := (*parsedPackageJSON)[key]
	if ok {
		dependencies := (*parsedPackageJSON)[key].(map[string]interface{})
		dependencies[module] = version
		(*parsedPackageJSON)[key] = dependencies
	}
}
