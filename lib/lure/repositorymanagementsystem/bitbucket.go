package repositorymanagementsystem

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

	"github.com/coveooss/lure/lib/lure/log"
	"github.com/coveooss/lure/lib/lure/project"
	"github.com/coveooss/lure/lib/lure/vcs"

	"github.com/sethgrid/pester"
)

type BitBucket struct {
	URL            string
	apiURL         string
	authentication vcs.Authentication
}

type pullRequestList struct {
	Next        string        `json:"next"`
	PullRequest []PullRequest `json:"values"`
}

type branch struct {
	Name string `json:"name"`
}

type source struct {
	Branch branch `json:"branch"`
}

type dest struct {
	Branch branch `json:"branch"`
}

type user struct {
	Uuid string `json:"uuid"`
}

func New(authentication vcs.Authentication, project project.Project) BitBucket {
	return BitBucket{
		URL:            "https://bitbucket.org/" + project.Owner + "/" + project.Name,
		apiURL:         "https://api.bitbucket.org/2.0/repositories",
		authentication: authentication,
	}
}

func (bitbucket BitBucket) GetURL() string {
	return bitbucket.URL
}

func (bitbucket BitBucket) GetPullRequests(username string, repoSlug string, ignoreDeclinedPRs bool) ([]PullRequest, error) {

	log.Logger.Info("Retrieving pull requests")

	acceptedStates := "state=OPEN"
	if !ignoreDeclinedPRs {
		acceptedStates += "&state=DECLINED"
	}

	bitBucketPath := fmt.Sprintf("/%s/%s/pullrequests/?%s", username, repoSlug, acceptedStates)

	prRequest, _ := bitbucket.createApiRequest("GET", bitBucketPath, nil)
	prRequest.Header.Add("Content-Type", "application/json")

	var list pullRequestList

	tmpList, e := bitbucket.getPRRequest(prRequest)
	if e != nil {
		log.Logger.Error(e.Error())
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
			tmpList, e = bitbucket.getPRRequest(prRequest)
			if e != nil {
				log.Logger.Error(e.Error())
				return nil, e
			}
			list.PullRequest = append(list.PullRequest, tmpList.PullRequest...)
		}
	}
	log.Logger.Infof("Found %d PRs.", len(list.PullRequest))

	return list.PullRequest, nil
}

func (bitbucket BitBucket) getDefaultReviewers(username string, repoSlug string) ([]user, error) {

	bitBucketPath := fmt.Sprintf("/%s/%s/default-reviewers", username, repoSlug)

	request, _ := bitbucket.createApiRequest("GET", bitBucketPath, nil)
	request.Header.Add("Content-Type", "application/json")

	client := getHTTPClient()
	resp, err := client.Do(request)

	if err != nil || resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Logger.Error("Error getting default reviewers", client.LogString())
		return nil, errors.New("Something went wrong getting default reviewers, got status code " + resp.Status)
	}

	type GetDefaultReviewers struct {
		Values []user `json:"values"`
	}
	var jsonresp GetDefaultReviewers
	json.NewDecoder(resp.Body).Decode(&jsonresp)

	defer resp.Body.Close()

	log.Logger.Tracef("Getting '%s' default reviewers returned %d: %d.", request.URL, resp.StatusCode, len(jsonresp.Values))

	return jsonresp.Values, nil
}

func (bitbucket BitBucket) getPRRequest(prRequest *http.Request) (*pullRequestList, error) {
	client := getHTTPClient()
	resp, err := client.Do(prRequest)

	if err != nil || resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Logger.Error("Error getting PR Requests", client.LogString())
		return nil, errors.New("Something went wrong getting PR, got status code " + resp.Status)
	}

	var prList pullRequestList
	json.NewDecoder(resp.Body).Decode(&prList)

	defer resp.Body.Close()

	log.Logger.Tracef("Getting '%s' PR returned %d.", prRequest.URL, resp.StatusCode)
	return &prList, nil
}

func (bitbucket BitBucket) CreatePullRequest(sourceBranch string, destBranch string, owner string, repo string, title string, description string, useDefaultReviewers bool) error {
	reviewers := []user{}
	if useDefaultReviewers {
		reviewers, _ = bitbucket.getDefaultReviewers(owner, repo)
	}

	pr := PullRequest{
		Title:       title,
		Description: description,
		Source: source{
			Branch: branch{
				Name: sourceBranch,
			},
		},
		Dest: dest{
			Branch: branch{
				Name: destBranch,
			},
		},
		CloseSourceBranch: true,
		Reviewers:         reviewers,
	}

	buf := &bytes.Buffer{}
	json.NewEncoder(buf).Encode(&pr)

	prRequest, err := bitbucket.createApiRequest("POST", fmt.Sprintf("/%s/%s/pullrequests/", owner, repo), buf)
	if err != nil {
		log.Logger.Error("Could not create a pull request")
		return err
	}

	prRequest.Header.Add("Content-Type", "application/json")

	log.Logger.Tracef("%v", prRequest)

	client := getHTTPClient()
	resp, err := client.Do(prRequest)

	if err != nil {
		log.Logger.Error("Error getting PR Requests", client.LogString())
		return err
	}

	defer resp.Body.Close()

	io.Copy(os.Stdout, resp.Body)

	return nil
}

func (bitbucket BitBucket) DeclinePullRequest(username string, repoSlug string, pullRequestID int) error {

	bitBucketPath := fmt.Sprintf("/%s/%s/pullrequests/%d/decline", username, repoSlug, pullRequestID)
	prRequest, err := bitbucket.createApiRequest("POST", bitBucketPath, strings.NewReader("{}"))
	if err != nil {
		log.Logger.Error("Could not decline pull request")
		return err
	}

	prRequest.Header.Add("Content-Type", "application/json")

	log.Logger.Tracef("%v", prRequest)

	client := getHTTPClient()
	resp, err := client.Do(prRequest)

	if err != nil {
		log.Logger.Error("Error declining PR Request", client.LogString())
		return err
	}

	defer resp.Body.Close()

	io.Copy(os.Stdout, resp.Body)

	return nil
}

func (bitbucket BitBucket) createApiRequest(method string, path string, body io.Reader) (*http.Request, error) {
	url := bitbucket.authentication.AuthenticateURL(bitbucket.apiURL + path)

	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	bitbucket.authentication.AuthenticateHTTPRequest(request.Header)

	return request, err
}

func getHTTPClient() *pester.Client {
	client := pester.New()
	client.MaxRetries = 10
	client.Backoff = pester.ExponentialBackoff
	client.RetryOnHTTP429 = true
	client.KeepLog = true
	return client
}
