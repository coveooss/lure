package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os/exec"
	"regexp"

	"github.com/blang/semver"
	"github.com/k0kubun/pp"
)

type packageJSON map[string]interface{}

func npmOutdated(path string) []moduleVersion {
	cmd := exec.Command("npm", "outdated")
	var out bytes.Buffer
	var errStrm bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errStrm
	cmd.Dir = path
	cmd.Run()

	reader := bytes.NewReader(out.Bytes())
	scanner := bufio.NewScanner(reader)

	npmRegex, _ := regexp.Compile(`([^\s]+)\s+([^\s]+)\s+([^\s]+)\s+([^\s]+)\s*`)

	lineIndex := 0

	version := make([]moduleVersion, 0, 0)
	for scanner.Scan() {
		if lineIndex != 0 {
			result := npmRegex.FindStringSubmatch(scanner.Text())
			mv := moduleVersion{
				Type: "npm",
				Module:  result[1],
				Wanted:  result[3],
				Current: result[2],
				Latest:  result[4],
			}

			wantedVersion, _ := semver.Parse(mv.Wanted)
			latestVersion, _ := semver.Parse(mv.Latest)

			if wantedVersion.LT(latestVersion) {
				pp.Printf("Including NPM version %s", mv)
				version = append(version, mv)
			}
		}
		lineIndex++
	}

	return version
}

func readPackageJSON(dir string, module string, version string) error {
	packageJSONBuffer, _ := ioutil.ReadFile(dir + "/package.json")
	var parsedPackageJSON packageJSON

	json.Unmarshal(packageJSONBuffer, &parsedPackageJSON)

	updateJSON(&parsedPackageJSON, "dependencies", module, version)
	updateJSON(&parsedPackageJSON, "devDependencies", module, version)
	updateJSON(&parsedPackageJSON, "optionalDependencies", module, version)

	updatedJSON, _ := json.MarshalIndent(&parsedPackageJSON, "", "  ")
	ioutil.WriteFile(dir+"/package.json", updatedJSON, 0770)

	return nil
}

func updateJSON(parsedPackageJSON *packageJSON, key string, module string, version string) {
	_, ok := (*parsedPackageJSON)[key]
	if ok {
		dependencies := (*parsedPackageJSON)[key].(map[string]interface{})
		dependencies[module] = version
		(*parsedPackageJSON)[key] = dependencies
	}
}

