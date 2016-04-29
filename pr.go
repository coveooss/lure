package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/k0kubun/pp"
)

type Branch struct {
	Name string `json:"name"`
}

type Source struct {
	Branch Branch `json:"branch"`
}

type PullRequest struct {
	Title             string `json:"title"`
	Description       string `json:"description"`
	Source            Source `json:"source"`
	CloseSourceBranch bool   `json:"close_source_branch"`
}

func potato() {
	createPullRequest(
		"lure-update-dep",
		"hmLf9fPPcQ4u0oBIwe3O2BvyqrMk3lHO9bLZ-fq73PC654R7hXrBu68y_Q6s_5gDBO6eafjZxVnlzC_Ogss=",
		"pastjean",
		"dummy",
		"react",
		"15.0.1")
}

func createPullRequest(branch string, token string, owner string, repo string, module string, version string) {
	pr := PullRequest{
		Title:       fmt.Sprintf("Update %s to version %s", module, version),
		Description: fmt.Sprintf("%s version %s is now available! Please update.", module, version),
		Source: Source{
			Branch: Branch{
				Name: branch,
			},
		},
		CloseSourceBranch: true,
	}

	url := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/%s/pullrequests/", owner, repo)

	buf := &bytes.Buffer{}
	json.NewEncoder(buf).Encode(&pr)

	prRequest, _ := http.NewRequest("POST", url, buf)
	prRequest.Header.Add("Content-Type", "application/json")
	prRequest.Header.Add("Authorization", "Bearer "+token)

	resp, _ := http.DefaultClient.Do(prRequest)

	body, _ := ioutil.ReadAll(resp.Body)
	pp.Println(string(body))
}
