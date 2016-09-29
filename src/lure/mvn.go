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

	//mvnRegex, _ := regexp.Compile(`\s*\[INFO\]\s+((?:[a-zA-Z0-9$_-]\.?))+:((?:[a-zA-Z0-9$_-]\.?))\s+\.{34}\s+((?:[a-zA-Z0-9$_-]\.?))\s+->\s+((?:[a-zA-Z0-9$_-]\.?))\s*`)
	mvnRegex, _ := regexp.Compile(`\s*\[INFO\]\s+((?:[a-zA-Z0-9$_-]\.?)+):((?:[a-zA-Z0-9$_-]\.?)+)\s+\.+\s+((?:[a-zA-Z0-9$_-]\.?)+)\s+->\s+((?:[a-zA-Z0-9$_-]\.?)+)\s*`)

	version := make([]moduleVersion, 0, 0)
	for scanner.Scan() {
		result := mvnRegex.FindStringSubmatch(scanner.Text())
		fmt.Printf("> %s - %q\n", scanner.Text(), result)
		if result != nil {
			mv := moduleVersion{
				Module:  result[1] + ":" + result[2],
				Current: result[3],
				Latest:  result[4],
			}

			pp.Println(mv)
			version = append(version, mv)
		}
	}

	return version
}

func mvnUpdateDep(path string, dependency string, version string) error {
	return execute(path, "mvn", "versions:use-dep-version", "-Dincludes="+dependency, "-DdepVersion="+version)
}
