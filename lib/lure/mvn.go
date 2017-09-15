package lure

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"log"
	"os/exec"
	"regexp"

	"fmt"
	"io/ioutil"
	"os"
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
	var modulePropertyMap map[string]string = getModulePropertyMap(path)

	for scanner.Scan() {
		log.Printf("> %s\n", scanner.Text())
		packageName := mvnPackageRegex.FindStringSubmatch(scanner.Text())
		if packageName != nil {
			lastPackage = packageName
		}

		packageVersion := mvnVersionRegex.FindStringSubmatch(scanner.Text())
		if packageVersion != nil {

			log.Printf(">%q - %q\n", packageName, packageVersion)
			mv := moduleVersion{
				Type: "maven",
				Module:  lastPackage[1] + ":" + lastPackage[2],
				Current: packageVersion[1],
				Wanted:  packageVersion[2],
				Latest:  packageVersion[2],
				Name:    modulePropertyMap[lastPackage[1] + ":" + lastPackage[2]],
			}
			log.Println(mv)
			version = append(version, mv)
		}
	}

	return version
}

func getModulePropertyMap(path string) map[string]string {
	var moduleProperties map[string]string = make(map[string]string)

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

	for _, folder := range folders {
		xmlFile, err := os.Open(folder + "/pom.xml")
		if err != nil {
			fmt.Println("Error opening file:", err)
			break;
		}
		defer xmlFile.Close()

		b, _ := ioutil.ReadAll(xmlFile)

		var mvnProject project
		xml.Unmarshal(b, &mvnProject)

		for _, dep := range mvnProject.Dependencies {
			if isProperty.MatchString(dep.Version)  {
				moduleProperties[(dep.GroupId + ":" + dep.ArtifactId)] = strings.TrimRight(strings.TrimLeft(dep.Version, "${"), "}")
			}
		}
	}
	return moduleProperties
}

type project struct {
	ModelVersion string `xml:"modelVersion"`
	Properties []struct {
		name xml.Name
		value string
	} `xml:"properties"`
	Dependencies []struct {
		ArtifactId string `xml:"artifactId"`
		GroupId string `xml:"groupId"`
		Version string `xml:"version"`
	} `xml:"dependencies>dependency"`
}

type property struct {
}


func mvnUpdateDep(path string, moduleVersion moduleVersion) (bool, error) { //dependency string, version string)
	dependency := moduleVersion.Module
	version := moduleVersion.Latest

	hasUpdate := false
	var err error

	if moduleVersion.Name != "" {
	//list all folder with pom.xml
	cmd := exec.Command("mvn",  "-q", "--also-make", "exec:exec", "-Dexec.executable=pwd")
	var out bytes.Buffer
	var stree bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stree
	cmd.Dir = path
	err := cmd.Run()

	if err != nil {
		log.Println(err)
		log.Fatal(err)
	}

	reader := bytes.NewReader(out.Bytes())
	scanner := bufio.NewScanner(reader)

	var folders []string
	for scanner.Scan() {
		log.Printf(scanner.Text())
		folders = append(folders, scanner.Text())
	}

	for _, folder := range folders {
		xmlFile, err := os.Open(folder + "/pom.xml")
		if err != nil {
			log.Println("Error opening file:", err)
			return false, err
		}
		defer xmlFile.Close()

		b, _ := ioutil.ReadAll(xmlFile)

		var mvnProject project
		xml.Unmarshal(b, &mvnProject)

			for _, property := range mvnProject.Properties {
				if property.name.Local == moduleVersion.Name {
					log.Println("%s : %s : %s", folder, property.name.Local, property.value)
					var propertyToReplace = strings.TrimRight(strings.TrimLeft(moduleVersion.Name, "${"), "}")

				for _, folder2 := range folders {
					xmlFile, err := os.Open(folder2 + "/pom.xml")
					if err != nil {
						log.Println("Error opening file:", err)
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
								propertyToReplace+ ">"+ moduleVersion.Current+
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
    } else {
    	var autoUpdateResult string
        autoUpdateResult, err = Execute(path, "mvn", "org.codehaus.mojo:versions-maven-plugin:2.4:use-dep-version", "-Dincludes="+dependency, "-DdepVersion="+version)
	if strings.Contains(autoUpdateResult, fmt.Sprintf("Updated %s:jar:%s to version %s", dependency, moduleVersion.Current, version)) == true {
		hasUpdate = true
	}
    }

	if hasUpdate == true {
		log.Printf("Updated %s:%s:jar:%s to version %s",  moduleVersion.Module, dependency, moduleVersion.Current, version)
	}

	return hasUpdate, err
}
