package main

import (
	"log"
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

type PullRequestList struct {
	PullRequest []PullRequest `json:"values"`
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

var apiURI = "https://api.bitbucket.org/2.0/repositories"

func getPullRequests(token string, username string, repoSlug string) []PullRequest {
	//Get Open PR
	url := fmt.Sprintf("%s/%s/%s/pullrequests/", apiURI, username, repoSlug)

	prRequest, _ := http.NewRequest("GET", url, nil)
	prRequest.Header.Add("Content-Type", "application/json")
	prRequest.Header.Add("Authorization", "Bearer " + token)

	//defer prRequest.Body.Close()

	resp, _ := http.DefaultClient.Do(prRequest)

	//body, _ := ioutil.ReadAll(resp.Body)
	//pp.Println(string(body))
	var list PullRequestList
	json.NewDecoder(resp.Body).Decode(&list)
	return list.PullRequest
}

func createPullRequest(branch string, token string, owner string, repo string, title string, description string) {
	pr := PullRequest{
		Title:       title,
		Description: description,
		Source: Source{
			Branch: Branch{
				Name: branch,
			},
		},
		CloseSourceBranch: true,
	}

	url := fmt.Sprintf("%s/%s/%s/pullrequests/", apiURI, owner, repo)

	buf := &bytes.Buffer{}
	json.NewEncoder(buf).Encode(&pr)

	prRequest, _ := http.NewRequest("POST", url, buf)
	prRequest.Header.Add("Content-Type", "application/json")
	prRequest.Header.Add("Authorization", "Bearer "+token)

	log.Printf("%s\n", prRequest)
	resp, _ := http.DefaultClient.Do(prRequest)

	body, _ := ioutil.ReadAll(resp.Body)
	pp.Println(string(body))
}
