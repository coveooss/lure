package lure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
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

	resp, e := getPRRequest(prRequest)
	json.NewDecoder(resp.Body).Decode(&tmpList)
	list.PullRequest = append(list.PullRequest, tmpList.PullRequest...)

	if tmpList.Next != "" {

		for tmpList.Next != "" && len(tmpList.PullRequest) != 0 {
			queryParams, _ := url.ParseQuery(tmpList.Next)
			nextQueryParams := prRequest.URL.Query()
			nextQueryParams.Set("page", queryParams.Get("page"))
			prRequest.URL.RawQuery = nextQueryParams.Encode()
			tmpList.Next = "" //Reset
			resp, e = getPRRequest(prRequest)
			json.NewDecoder(resp.Body).Decode(&tmpList)
			list.PullRequest = append(list.PullRequest, tmpList.PullRequest...)
		}
	}
	log.Printf("Found %d PRs.", len(list.PullRequest))

	if e != nil {
		log.Println("error: " + e.Error())
	}

	list.PullRequest = append(list.PullRequest, tmpList.PullRequest...)

	return list.PullRequest
}
func getPRRequest(prRequest *http.Request) (*http.Response, error) {
	resp, e := http.DefaultClient.Do(prRequest)
	for !(resp.StatusCode == 200 && resp.StatusCode < 300) {
		log.Printf("Getting '%s' PR returned %d. Retrying...", prRequest.URL, resp.StatusCode)
		resp, e = http.DefaultClient.Do(prRequest)
		time.Sleep(time.Second)
	}
	log.Printf("Getting '%s' PR returned %d.", prRequest.URL, resp.StatusCode)
	return resp, e
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

	log.Printf("%v\n", prRequest)

	resp, err := http.DefaultClient.Do(prRequest)
	if err != nil {
		return err
	}

	io.Copy(os.Stdout, resp.Body)
	return nil
}
