package main

import (
	"os"
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

type Dest struct {
	Branch Branch `json:"branch"`
}

type PullRequest struct {
	Title             string `json:"title"`
	Description       string `json:"description"`
	Source            Source `json:"source"`
	Dest              Dest   `json:"dest"`
	CloseSourceBranch bool   `json:"close_source_branch"`
}

type PullRequestList struct {
	PullRequest []PullRequest `json:"values"`
}

var apiURI = "https://api.bitbucket.org/2.0/repositories"

func getPullRequests(token string, username string, repoSlug string) []PullRequest {

	acceptedStates := "state=OPEN"
	if (os.Getenv("IGNORE_DECLINED_PR") != "1") {
		acceptedStates += "&state=DECLINED"
	}

	url := fmt.Sprintf("%s/%s/%s/pullrequests/?%s", apiURI, username, repoSlug, acceptedStates)

	prRequest, _ := http.NewRequest("GET", url, nil)
	prRequest.Header.Add("Content-Type", "application/json")
	prRequest.Header.Add("Authorization", "Bearer " + token)

	resp, _ := http.DefaultClient.Do(prRequest)

	var list PullRequestList
	json.NewDecoder(resp.Body).Decode(&list)
	return list.PullRequest
}

func createPullRequest(token string, sourceBranch string, destBranch string, owner string, repo string, title string, description string) (error) {
	pr := PullRequest{
		Title:       title,
		Description: description,
		Source: Source{
			Branch: Branch{
				Name: sourceBranch,
			},
		},
		Dest: Dest{
			Branch: Branch{
				Name: destBranch,
			},
		},
		CloseSourceBranch: true,
	}

	url := fmt.Sprintf("%s/%s/%s/pullrequests/", apiURI, owner, repo)

	buf := &bytes.Buffer{}
	json.NewEncoder(buf).Encode(&pr)

	prRequest, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return err
	}

	prRequest.Header.Add("Content-Type", "application/json")
	prRequest.Header.Add("Authorization", "Bearer "+token)

	log.Printf("%s\n", prRequest)
	resp, err := http.DefaultClient.Do(prRequest)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	pp.Println(string(body))

	return nil
}
