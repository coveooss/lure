package lure

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

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

type User struct {
	Uuid string `json:"uuid"`
}

type PullRequest struct {
	ID                int    `json:"id"`
	Title             string `json:"title"`
	Description       string `json:"description"`
	Source            Source `json:"source"`
	Dest              Dest   `json:"destination"`
	CloseSourceBranch bool   `json:"close_source_branch"`
	State             string `json:"state"`
	Reviewers         []User `json:"reviewers"`
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

func getPullRequests(auth Authentication, username string, repoSlug string, ignoreDeclinedPRs bool) ([]PullRequest, error) {

	acceptedStates := "state=OPEN"
	if !ignoreDeclinedPRs {
		acceptedStates += "&state=DECLINED"
	}

	bitBucketPath := fmt.Sprintf("/%s/%s/pullrequests/?%s", username, repoSlug, acceptedStates)

	prRequest, _ := createApiRequest(auth, "GET", bitBucketPath, nil)
	prRequest.Header.Add("Content-Type", "application/json")

	var list PullRequestList

	tmpList, e := getPRRequest(prRequest)
	if e != nil {
		Logger.Error(e.Error())
		return nil, e
	}
	list.PullRequest = append(list.PullRequest, tmpList.PullRequest...)

	if tmpList.Next != "" {

		for tmpList.Next != "" && len(tmpList.PullRequest) != 0 {
			queryParams, _ := url.ParseQuery(tmpList.Next)
			nextQueryParams := prRequest.URL.Query()
			nextQueryParams.Set("page", queryParams.Get("page"))
			prRequest.URL.RawQuery = nextQueryParams.Encode()
			tmpList.Next = "" //Reset
			tmpList, e = getPRRequest(prRequest)
			if e != nil {
				Logger.Error(e.Error())
				return nil, e
			}
			list.PullRequest = append(list.PullRequest, tmpList.PullRequest...)
		}
	}
	Logger.Infof("Found %d PRs.", len(list.PullRequest))

	return list.PullRequest, nil
}

func getDefaultReviewers(auth Authentication, username string, repoSlug string) ([]User, error) {

	bitBucketPath := fmt.Sprintf("/%s/%s/default-reviewers", username, repoSlug)

	request, _ := createApiRequest(auth, "GET", bitBucketPath, nil)
	request.Header.Add("Content-Type", "application/json")

	client := getHTTPClient()
	resp, err := client.Do(request)

	if err != nil || resp.StatusCode < 200 || resp.StatusCode > 299 {
		Logger.Error("Error getting default reviewers", client.LogString())
		return nil, errors.New("Something went wrong getting default reviewers, got status code " + resp.Status)
	}

	type GetDefaultReviewers struct {
		Values []User `json:"values"`
	}
	var jsonresp GetDefaultReviewers
	json.NewDecoder(resp.Body).Decode(&jsonresp)

	defer resp.Body.Close()

	Logger.Tracef("Getting '%s' default reviewers returned %d: %d.", request.URL, resp.StatusCode, len(jsonresp.Values))

	return jsonresp.Values, nil
}

func getPRRequest(prRequest *http.Request) (*PullRequestList, error) {
	client := getHTTPClient()
	resp, err := client.Do(prRequest)

	if err != nil || resp.StatusCode < 200 || resp.StatusCode > 299 {
		Logger.Error("Error getting PR Requests", client.LogString())
		return nil, errors.New("Something went wrong getting PR, got status code " + resp.Status)
	}

	var prList PullRequestList
	json.NewDecoder(resp.Body).Decode(&prList)

	defer resp.Body.Close()

	Logger.Tracef("Getting '%s' PR returned %d.", prRequest.URL, resp.StatusCode)
	return &prList, nil
}

func createPullRequest(auth Authentication, sourceBranch string, destBranch string, owner string, repo string, title string, description string, useDefaultReviewers bool) error {
	reviewers := []User{}
	if useDefaultReviewers {
		reviewers, _ = getDefaultReviewers(auth, owner, repo)
	}

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
		Reviewers:         reviewers,
	}

	buf := &bytes.Buffer{}
	json.NewEncoder(buf).Encode(&pr)

	prRequest, err := createApiRequest(auth, "POST", fmt.Sprintf("/%s/%s/pullrequests/", owner, repo), buf)
	if err != nil {
		Logger.Error("Could not create a pull request")
		return err
	}

	prRequest.Header.Add("Content-Type", "application/json")

	Logger.Tracef("%v", prRequest)

	client := getHTTPClient()
	resp, err := client.Do(prRequest)

	if err != nil {
		Logger.Error("Error getting PR Requests", client.LogString())
		return err
	}

	defer resp.Body.Close()

	io.Copy(os.Stdout, resp.Body)

	return nil
}

func declinePullRequest(auth Authentication, username string, repoSlug string, pullRequestID int) error {

	bitBucketPath := fmt.Sprintf("/%s/%s/pullrequests/%d/decline", username, repoSlug, pullRequestID)
	prRequest, err := createApiRequest(auth, "POST", bitBucketPath, strings.NewReader("{}"))
	if err != nil {
		Logger.Error("Could not decline pull request")
		return err
	}

	prRequest.Header.Add("Content-Type", "application/json")

	Logger.Tracef("%v", prRequest)

	client := getHTTPClient()
	resp, err := client.Do(prRequest)

	if err != nil {
		Logger.Error("Error declining PR Request", client.LogString())
		return err
	}

	defer resp.Body.Close()

	io.Copy(os.Stdout, resp.Body)

	return nil
}

func getHTTPClient() *pester.Client {
	client := pester.New()
	client.MaxRetries = 10
	client.Backoff = pester.ExponentialBackoff
	client.RetryOnHTTP429 = true
	client.KeepLog = true
	return client
}
