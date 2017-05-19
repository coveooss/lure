package main

import (
	"bufio"
	"bytes"
	"log"
	"os/exec"
	"regexp"

	"github.com/k0kubun/pp"
	"fmt"
)

func mvnOutdated(path string) []moduleVersion {
	cmd := exec.Command("mvn", "versions:display-dependency-updates")
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

func mvnUpdateDep(path string, dependency string, version string) error {
	_, err := execute(path, "mvn", "versions:use-dep-version", "-Dincludes="+dependency, "-DdepVersion="+version)
	return err
}
