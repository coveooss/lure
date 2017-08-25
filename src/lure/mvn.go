package main

import (
	"bufio"
	"bytes"
	"log"
	"os/exec"
	"regexp"
	"encoding/xml"

	"github.com/k0kubun/pp"
	"fmt"
	"os"
	"io/ioutil"
	"strings"
	"launchpad.net/xmlpath"
)

func mvnOutdated(path string) []moduleVersion {
	cmd := exec.Command("mvn", "versions:display-dependency-updates", "-DprocessDependencyManagement=false")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Dir = path
	err := cmd.Run()

	if err != nil {
		log.Fatal(err)
	}

	reader := bytes.NewReader(out.Bytes())
	scanner := bufio.NewScanner(reader)

	mvnPackageRegex, _ := regexp.Compile(`\s*\[INFO\]\s+((?:[a-zA-Z0-9$_-]\.?)+):((?:[a-zA-Z0-9$_-]\.?)+)\s+`)
	mvnVersionRegex, _ := regexp.Compile(`((?:[a-zA-Z0-9$_-]\.?)+)\s+->\s+((?:[a-zA-Z0-9$_-]\.?)+)\s*`)

	version := make([]moduleVersion, 0, 0)
	var lastPackage []string
	for scanner.Scan() {
		fmt.Printf("> %s\n", scanner.Text())
		packageName := mvnPackageRegex.FindStringSubmatch(scanner.Text())
		if(packageName != nil) {
			lastPackage = packageName
		}

		packageVersion := mvnVersionRegex.FindStringSubmatch(scanner.Text())
		if (packageVersion != nil) {

			fmt.Printf(">%q - %q\n", packageName, packageVersion)
			mv := moduleVersion{
				Type: "maven",
				Module:  lastPackage[1] + ":" + lastPackage[2],
				Current: packageVersion[1],
				Wanted:  packageVersion[2],
				Latest:  packageVersion[2],
			}
			pp.Println(mv)
			version = append(version, mv)
		}
	}

	return version
}

type project struct {
	ModelVersion string `xml:"modelVersion"`
	Dependencies []struct {
		ArtifactId string `xml:"artifactId"`
		GroupId string `xml:"groupId"`
		Version string `xml:"version"`
	} `xml:"dependencies>dependency"`
}

type property struct {

}


func mvnUpdateDep(path string, mver moduleVersion) (bool, error) { //dependency string, version string)
	dependency := mver.Module
	version := mver.Latest

	hasUpdate := false

	//list all folder with pom.xml
	cmd := exec.Command("mvn",  "-q", "--also-make", "exec:exec", "-Dexec.executable=pwd")
	var out bytes.Buffer
	var stree bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stree
	cmd.Dir = path
	err := cmd.Run()

	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}

	reader := bytes.NewReader(out.Bytes())
	scanner := bufio.NewScanner(reader)

	var folders []string
	for scanner.Scan() {
		fmt.Printf(scanner.Text())
		folders = append(folders, scanner.Text())
	}

	isProperty, _ := regexp.Compile(`\$\{[\w.-]+}`)
	var propertyToReplace string

	for _, folder := range folders {
		xmlFile, err := os.Open(folder + "/pom.xml")
		if err != nil {
			fmt.Println("Error opening file:", err)
			return false, err
		}
		defer xmlFile.Close()

		b, _ := ioutil.ReadAll(xmlFile)

		var mvnProject project
		xml.Unmarshal(b, &mvnProject)

		for _, dep := range mvnProject.Dependencies {
			if isProperty.MatchString(dep.Version) && dependency == (dep.GroupId + ":" + dep.ArtifactId) {
				fmt.Println("%s : %s : %s", folder, dep.ArtifactId, dep.Version)
				propertyToReplace = strings.TrimRight(strings.TrimLeft(dep.Version, "${"), "}")

				for _, folder2 := range folders {
					xmlFile, err := os.Open(folder2 + "/pom.xml")
					if err != nil {
						fmt.Println("Error opening file:", err)
						return false, err
					}
					defer xmlFile.Close()

					path := xmlpath.MustCompile("/project/properties/" +
						propertyToReplace)
					root, err := xmlpath.Parse(xmlFile)
					if err != nil {
						log.Fatal(err)
					}
					if _, ok := path.String(root); ok {
						b, _ := ioutil.ReadFile(folder2 + "/pom.xml")
						newContent := strings.Replace(string(b), "<" +
							propertyToReplace + ">" + mver.Current +
							"</" + propertyToReplace + ">",
							"<" + propertyToReplace + ">" +
								version +
							"</" + propertyToReplace + ">", -1)

						err = ioutil.WriteFile(folder2 + "/pom.xml", []byte(newContent), 0)
						if err != nil {
							panic(err)
						}
						hasUpdate = true
					}
				}
			}
		}
	}
	autoUpdateResult, err := execute(path, "mvn", "org.codehaus.mojo:versions-maven-plugin:2.4:use-dep-version", "-Dincludes="+dependency, "-DdepVersion="+version)

	if strings.Contains(autoUpdateResult, fmt.Sprintf("Updated %s:%s:jar:%s to version %s",  mver.Module, dependency, mver.Current, version)) == true {
		hasUpdate = true
	}

	return hasUpdate, err
}
