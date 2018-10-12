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

	"github.com/sethgrid/pester"
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
		return nil, err
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

	tmpList, e := getPRRequest(prRequest)
	list.PullRequest = append(list.PullRequest, tmpList.PullRequest...)

	if tmpList.Next != "" {

		for tmpList.Next != "" && len(tmpList.PullRequest) != 0 {
			queryParams, _ := url.ParseQuery(tmpList.Next)
			nextQueryParams := prRequest.URL.Query()
			nextQueryParams.Set("page", queryParams.Get("page"))
			prRequest.URL.RawQuery = nextQueryParams.Encode()
			tmpList.Next = "" //Reset
			tmpList, e = getPRRequest(prRequest)
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
func getPRRequest(prRequest *http.Request) (*PullRequestList, error) {
	client := getHTTPClient()
	resp, err := client.Do(prRequest)

	if err != nil {
		log.Println("Error getting PR Requests", client.LogString())
		return nil, err
	}

	var prList PullRequestList
	json.NewDecoder(resp.Body).Decode(&prList)

	defer resp.Body.Close()

	log.Printf("Getting '%s' PR returned %d.", prRequest.URL, resp.StatusCode)
	return &prList, nil
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
		log.Println("Could not create a pull request")
		return err
	}

	prRequest.Header.Add("Content-Type", "application/json")

	log.Printf("%v\n", prRequest)

	client := getHTTPClient()
	resp, err := client.Do(prRequest)

	if err != nil {
		log.Println("Error getting PR Requests", client.LogString())
		return err
	}

	defer resp.Body.Close()

	io.Copy(os.Stdout, resp.Body)

	return nil
}

func getHTTPClient() *pester.Client {
	client := pester.New()
	client.MaxRetries = 5
	client.Backoff = pester.ExponentialBackoff
	return client
}
