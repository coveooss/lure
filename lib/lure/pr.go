package lure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
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
	Next        string        `json:"next"`
	PullRequest []PullRequest `json:"values"`
}

var apiURI = "api.bitbucket.org/2.0/repositories"

func createApiRequest(auth Authentication, method string, path string, body io.Reader) (*http.Request, error) {
	var url = ""
	switch auth := auth.(type) {
	case UserPassAuth:
		url = fmt.Sprintf("https://%s:%s@%s%s", auth.Username, auth.Password, apiURI, path)
	default:
		url = fmt.Sprintf("https://%s%s", apiURI, path)
	}

	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return request, err
	}

	switch auth := auth.(type) {
	case TokenAuth:
		request.Header.Add("Authorization", "Bearer "+auth.Token)
	}

	return request, err
}

func getPullRequests(auth Authentication, username string, repoSlug string) []PullRequest {

	acceptedStates := "state=OPEN"
	if os.Getenv("IGNORE_DECLINED_PR") != "1" {
		acceptedStates += "&state=DECLINED"
	}

	bitBucketPath := fmt.Sprintf("/%s/%s/pullrequests/?%s", username, repoSlug, acceptedStates)

	prRequest, _ := createApiRequest(auth, "GET", bitBucketPath, nil)
	prRequest.Header.Add("Content-Type", "application/json")

	var list PullRequestList
	var tmpList PullRequestList

	resp, e := http.DefaultClient.Do(prRequest)
	json.NewDecoder(resp.Body).Decode(&tmpList)
	list.PullRequest = append(list.PullRequest, tmpList.PullRequest...)

	if tmpList.Next != "" {
		for tmpList.Next != "" && len(tmpList.PullRequest) != 0 {
			prRequest.URL, _ = url.Parse(tmpList.Next)
			tmpList.Next = "" //Reset
			resp, e = http.DefaultClient.Do(prRequest)
			json.NewDecoder(resp.Body).Decode(&tmpList)
			list.PullRequest = append(list.PullRequest, tmpList.PullRequest...)
		}
	}

	if e != nil {
		fmt.Println("error: " + e.Error())
	}

	list.PullRequest = append(list.PullRequest, tmpList.PullRequest...)

	return list.PullRequest
}

func createPullRequest(auth Authentication, sourceBranch string, destBranch string, owner string, repo string, title string, description string) error {
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

	prRequest, err := createApiRequest(auth, "POST", fmt.Sprintf("/%s/%s/pullrequests/", owner, repo), buf)
	if err != nil {
		return err
	}

	prRequest.Header.Add("Content-Type", "application/json")

	log.Printf("%s\n", prRequest)

	resp, err := http.DefaultClient.Do(prRequest)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println(string(body))
	return nil
}
