package npm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"

	"github.com/blang/semver"
	"github.com/coveooss/lure/lib/lure/log"
	"github.com/coveooss/lure/lib/lure/versionManager"
)

type packageJSON map[string]interface{}

func NpmOutdated(path string) []versionManager.ModuleVersion {
	const packageJSONDefaultFileName = "package.json"
	if _, err := os.Stat(path + packageJSONDefaultFileName); os.IsNotExist(err) {
		log.Logger.Infof(packageJSONDefaultFileName + " doesn't exist, skipping npm update")
		return make([]versionManager.ModuleVersion, 0, 0)
	}

	log.Logger.Infof("Running npm install")
	cmd := exec.Command("npm", "install")
	cmd.Dir = path
	err := cmd.Run()
	if err != nil {
		log.Logger.Errorf("Could not npm install: '%s'\n", err)
		return make([]versionManager.ModuleVersion, 0, 0)
	}

	cmd = exec.Command("npm", "outdated")
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

	version := make([]versionManager.ModuleVersion, 0, 0)
	for scanner.Scan() {
		if lineIndex != 0 {
			result := npmRegex.FindStringSubmatch(scanner.Text())
			mv := versionManager.ModuleVersion{
				Type:    "npm",
				Module:  result[1],
				Wanted:  result[3],
				Current: result[2],
				Latest:  result[4],
			}
			wantedVersion, _ := semver.Parse(mv.Wanted)
			latestVersion, _ := semver.Parse(mv.Latest)

			if wantedVersion.LT(latestVersion) {
				log.Logger.Infof("Including NPM version %s", mv)
				version = append(version, mv)
			}
		}
		lineIndex++
	}

	return version
}

func UpdateDependency(dir string, moduleToUpdate versionManager.ModuleVersion) (bool, error) {
	module := moduleToUpdate.Module
	version := moduleToUpdate.Latest
	packageJSONBuffer, _ := ioutil.ReadFile(dir + "/package.json")
	var parsedPackageJSON packageJSON

	json.Unmarshal(packageJSONBuffer, &parsedPackageJSON)

	updateJSON(&parsedPackageJSON, "dependencies", module, version)
	updateJSON(&parsedPackageJSON, "devDependencies", module, version)
	updateJSON(&parsedPackageJSON, "optionalDependencies", module, version)

	// json.Marshal HTML encode the characters. We need to use a custom encoder to fix that.
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	enc.Encode(parsedPackageJSON)
	updatedJSON := buf.Bytes()
	ioutil.WriteFile(dir+"/package.json", updatedJSON, 0770)

	return true, nil
}

func updateJSON(parsedPackageJSON *packageJSON, key string, module string, version string) {
	_, ok := (*parsedPackageJSON)[key]
	if ok {
		dependencies := (*parsedPackageJSON)[key].(map[string]interface{})

		// Only update the dependency if it exists already.
		if dependencies[module] != nil {
			// Check for version operators and reuse them
			operator := getRangeOperator(dependencies[module].(string))
			version = operator + version

			dependencies[module] = version
			(*parsedPackageJSON)[key] = dependencies
		}

	}
}

func getRangeOperator(version string) string {
	// https://docs.npmjs.com/misc/semver#x-ranges-12x-1x-12-
	r, _ := regexp.Compile("^(\\^|~).*")

	operators := r.FindStringSubmatch(version)

	if len(operators) > 0 {
		// The index zero is the whole match, index 1 is the first group match.
		return operators[1]
	}

	return ""
}
