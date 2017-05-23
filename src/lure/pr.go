package main

import (
	"os"
	"io"
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
	Dest              Dest   `json:"destination"`
	CloseSourceBranch bool   `json:"close_source_branch"`
}

type PullRequestList struct {
	PullRequest []PullRequest `json:"values"`
}

var apiURI = "api.bitbucket.org/2.0/repositories"

func createApiRequest(auth Authentication, method string, path string, body io.Reader) (*http.Request, error) {
	var url = ""
	switch auth := auth.(type) {
	case UserPassAuth:
		url = fmt.Sprintf("https://%s:%s@%s/%s", auth.username, auth.password, apiURI, path)
	default:
		url = fmt.Sprintf("https://%s/%s", apiURI, path)
	}

	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return request, err
	}

	switch auth := auth.(type) {
	case TokenAuth:
		request.Header.Add("Authorization", "Bearer " + auth.token)
	}

	return request, err
}

func getPullRequests(auth Authentication, username string, repoSlug string) []PullRequest {

	acceptedStates := "state=OPEN"
	if (os.Getenv("IGNORE_DECLINED_PR") != "1") {
		acceptedStates += "&state=DECLINED"
	}

	url := fmt.Sprintf("https://%s/%s/%s/pullrequests/?%s", apiURI, username, repoSlug, acceptedStates)

	prRequest, _ := createApiRequest(auth, "GET", url, nil)
	prRequest.Header.Add("Content-Type", "application/json")

	resp, _ := http.DefaultClient.Do(prRequest)

	var list PullRequestList
	json.NewDecoder(resp.Body).Decode(&list)
	return list.PullRequest
}

func createPullRequest(auth Authentication, sourceBranch string, destBranch string, owner string, repo string, title string, description string) (error) {
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

	buf := &bytes.Buffer{}
	json.NewEncoder(buf).Encode(&pr)

	prRequest, err := createApiRequest(auth, "POST", fmt.Sprintf("%s/%s/pullrequests/", owner, repo), buf)
	if err != nil {
		return err
	}

	prRequest.Header.Add("Content-Type", "application/json")

	log.Printf("%s\n", prRequest)
	if (os.Getenv("DRY_RUN") == "1") {
		log.Println("Running in DryRun mode, not doing the request")
	} else {
		resp, err := http.DefaultClient.Do(prRequest)
		if err != nil {
			return err
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		pp.Println(string(body))
	}
	return nil
}
