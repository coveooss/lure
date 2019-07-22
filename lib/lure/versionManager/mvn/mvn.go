package mvn

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"os/exec"
	"regexp"

	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/coveooss/lure/lib/lure/log"
	osUtils "github.com/coveooss/lure/lib/lure/os"
	"github.com/coveooss/lure/lib/lure/versionManager"
	"launchpad.net/xmlpath"
)

func MvnOutdated(path string) (error, []versionManager.ModuleVersion) {
	const pomDefaultFileName = "pom.xml"
	if !fileExists(path + pomDefaultFileName) {
		log.Logger.Info(pomDefaultFileName + " doesn't exist, skipping mvn update")
		return nil, make([]versionManager.ModuleVersion, 0, 0)
	}

	var cmd *exec.Cmd
	if fileExists("Rules.xml") {
		cmd = exec.Command("mvn", "-B", "versions:display-dependency-updates", "-DprocessDependencyManagement=false", "-Dmaven.version.rules=file:Rules.xml")
	} else {
		cmd = exec.Command("mvn", "-B", "versions:display-dependency-updates", "-DprocessDependencyManagement=false")
	}
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	cmd.Dir = path
	err := cmd.Run()

	if err != nil {
		log.Logger.Error("Error running mvn versions:display-dependency-updates")
		log.Logger.Error(out.String())
		log.Logger.Error(stderr.String())
		log.Logger.Error(err)
		return err, make([]versionManager.ModuleVersion, 0, 0)
	}

	reader := bytes.NewReader(out.Bytes())
	scanner := bufio.NewScanner(reader)

	mvnPackageRegex, _ := regexp.Compile(`\s*\[INFO\]\s+((?:[a-zA-Z0-9$_-]\.?)+):((?:[a-zA-Z0-9$_-]\.?)+)\s+`)
	mvnVersionRegex, _ := regexp.Compile(`((?:[a-zA-Z0-9$_-]\.?)+)\s+->\s+((?:[a-zA-Z0-9$_-]\.?)+)\s*`)

	version := make([]versionManager.ModuleVersion, 0, 0)
	var lastPackage []string
	var modulePropertyMap map[string]string = getModulePropertyMap(path)

	for scanner.Scan() {
		log.Logger.Tracef("> %s\n", scanner.Text())
		packageName := mvnPackageRegex.FindStringSubmatch(scanner.Text())
		if packageName != nil {
			lastPackage = packageName
		}

		packageVersion := mvnVersionRegex.FindStringSubmatch(scanner.Text())
		if packageVersion != nil {

			log.Logger.Tracef(">%q - %q\n", packageName, packageVersion)
			mv := versionManager.ModuleVersion{
				Type:    "maven",
				Module:  lastPackage[1] + ":" + lastPackage[2],
				Current: packageVersion[1],
				Wanted:  packageVersion[2],
				Latest:  packageVersion[2],
				Name:    modulePropertyMap[lastPackage[1]+":"+lastPackage[2]],
			}
			log.Logger.Trace(mv)
			version = append(version, mv)
		}
	}

	return nil, version
}

func getModulePropertyMap(path string) map[string]string {
	var moduleProperties map[string]string = make(map[string]string)

	cmd := exec.Command("mvn", "-B", "-q", "--also-make", "exec:exec", "-Dexec.executable=pwd")
	var out bytes.Buffer
	var stree bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stree
	cmd.Dir = path
	err := cmd.Run()

	if err != nil {
		log.Logger.Error("Error running mvn -q --also-make exec:exec -Dexec.executable=pwd")
		log.Logger.Error(err)
		log.Logger.Error(out.String())
		log.Logger.Error(stree.String())
		log.Logger.Fatal(err)
	}

	reader := bytes.NewReader(out.Bytes())
	scanner := bufio.NewScanner(reader)

	var folders []string
	for scanner.Scan() {
		log.Logger.Infof(scanner.Text())
		folders = append(folders, scanner.Text())
	}

	isProperty, _ := regexp.Compile(`\$\{[\w.-]+}`)

	for _, folder := range folders {
		xmlFile, err := os.Open(folder + "/pom.xml")
		if err != nil {
			log.Logger.Error("Error opening file:", err)
			break
		}
		defer xmlFile.Close()

		b, _ := ioutil.ReadAll(xmlFile)

		var mvnProject mvnProjectDef
		xml.Unmarshal(b, &mvnProject)

		for _, dep := range mvnProject.Dependencies {
			if isProperty.MatchString(dep.Version) {
				moduleProperties[(dep.GroupId + ":" + dep.ArtifactId)] = strings.TrimRight(strings.TrimLeft(dep.Version, "${"), "}")
			}
		}
	}
	return moduleProperties
}

type mvnProjectDef struct {
	ModelVersion string        `xml:"modelVersion"`
	Properties   PropertyArray `xml:"properties"`
	Dependencies []struct {
		ArtifactId string `xml:"artifactId"`
		GroupId    string `xml:"groupId"`
		Version    string `xml:"version"`
	} `xml:"dependencies>dependency"`
}

type PropertyArray struct {
	PropertyList []Property `xml:",any"`
}
type Property struct {
	XMLName xml.Name `xml:""`
	Value   string   `xml:",chardata"`
}

type property struct {
}

func UpdateDependency(path string, moduleVersion versionManager.ModuleVersion) (bool, error) { //dependency string, version string)
	dependency := moduleVersion.Module
	version := moduleVersion.Latest

	hasUpdate := false
	var err error

	if moduleVersion.Name != "" {
		//list all folder with pom.xml
		cmd := exec.Command("mvn", "-B", "-q", "--also-make", "exec:exec", "-Dexec.executable=pwd")
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		cmd.Dir = path
		err := cmd.Run()

		if err != nil {
			log.Logger.Error(stderr.String())
			log.Logger.Fatal(err)
		}

		reader := bytes.NewReader(out.Bytes())
		scanner := bufio.NewScanner(reader)

		var folders []string
		for scanner.Scan() {
			log.Logger.Info(scanner.Text())
			folders = append(folders, scanner.Text())
		}

		for _, folder := range folders {
			xmlFile, err := os.Open(folder + "/pom.xml")
			if err != nil {
				log.Logger.Println("Error opening file:", err)
				return false, err
			}
			defer xmlFile.Close()

			b, _ := ioutil.ReadAll(xmlFile)

			var mvnProject mvnProjectDef
			xml.Unmarshal(b, &mvnProject)

			for _, property := range mvnProject.Properties.PropertyList {
				if property.XMLName.Local == moduleVersion.Name {
					log.Logger.Infof("%s : %s : %s", folder, property.XMLName.Local, property.Value)
					var propertyToReplace = strings.TrimRight(strings.TrimLeft(moduleVersion.Name, "${"), "}")

					for _, folder2 := range folders {
						xmlFile, err := os.Open(folder2 + "/pom.xml")
						if err != nil {
							log.Logger.Error("Error opening file:", err)
							return false, err
						}
						defer xmlFile.Close()

						path := xmlpath.MustCompile("/project/properties/" +
							propertyToReplace)
						root, err := xmlpath.Parse(xmlFile)
						if err != nil {
							log.Logger.Fatal(err)
						}
						if _, ok := path.String(root); ok {
							b, _ := ioutil.ReadFile(folder2 + "/pom.xml")
							newContent := strings.Replace(string(b),
								"<"+propertyToReplace+">"+moduleVersion.Current+"</"+propertyToReplace+">",
								"<"+propertyToReplace+">"+version+"</"+propertyToReplace+">", -1)

							err = ioutil.WriteFile(folder2+"/pom.xml", []byte(newContent), 0)
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
		autoUpdateResult, err = osUtils.Execute(path, "mvn", "-B", "org.codehaus.mojo:versions-maven-plugin:2.4:use-dep-version", "-Dincludes="+dependency, "-DdepVersion="+version)
		if strings.Contains(autoUpdateResult, fmt.Sprintf("Updated %s:jar:%s to version %s", dependency, moduleVersion.Current, version)) == true {
			hasUpdate = true
		}
	}

	if hasUpdate == true {
		log.Logger.Infof("Updated %s:%s:jar:%s to version %s", moduleVersion.Module, dependency, moduleVersion.Current, version)
	}

	return hasUpdate, err
}

func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
