package repositorymanagementsystem

import (
	"context"
	"fmt"
	"github.com/coveooss/lure/lib/lure/log"
	"github.com/coveooss/lure/lib/lure/project"
	"github.com/coveooss/lure/lib/lure/vcs"
	"github.com/google/go-github/v32/github"
)



type GitHub struct {
	URL            string
	apiURL         string
	authentication vcs.Authentication
}


func (gh GitHub) GetURL() string {
	return gh.URL
}

func NewGitHub(authentication vcs.Authentication, project project.Project) GitHub {
	return GitHub{
		URL:            "https://github.com/" + project.Owner + "/" + project.Name,
		apiURL:         "https://api.github.com/" + project.Owner + "/" + project.Name,
		authentication: authentication,
	}
}


func (gh GitHub) CreatePullRequest(sourceBranch string, destBranch string, owner string, repo string, title string, description string, useDefaultReviewers bool) error {
	httpClient := gh.authentication.AuthenticateWithToken()
	client := github.NewClient(httpClient)

	newPR := github.NewPullRequest{
		Title:               &title,
		Head:                &destBranch,
		Base:                &sourceBranch,
		Body:                &description,
	}

	pr, _, err := client.PullRequests.Create(context.Background(), owner, repo, &newPR)

	if err != nil {
		log.Logger.Error("Error creating GitHub Pull Request")
		return err
	}

	log.Logger.Info(fmt.Sprintf("Created Pull Request %x", *pr.Number))

	return nil
}

func (gh GitHub) GetPullRequests(username string, repoSlug string, ignoreDeclinedPRs bool) ([]PullRequest, error) {
	httpClient := gh.authentication.AuthenticateWithToken()
	client := github.NewClient(httpClient)

	state := "open"
	if !ignoreDeclinedPRs {
		state = "all"
	}
	options := github.PullRequestListOptions{State: state}
	prs, _, err := client.PullRequests.List(context.Background(), username, repoSlug, &options)

	if err != nil {
		log.Logger.Error("Error listing GitHub Pull Requests")
		return nil, err
	}

	var pullRequests []PullRequest
	for _, pr := range prs {
		options := github.ListOptions{Page: 1, PerPage: 100}
		reviews, _, err := client.PullRequests.ListReviews(context.Background(), username, repoSlug, *pr.Number, &options)

		if err != nil {
			log.Logger.Error("Error getting GitHub Pull Request")
			return nil, err
		}

		var reviewers []user
		for _, review := range reviews {
			reviewers = append(reviewers, user{*review.User.Login})
		}

		pullRequests = append(pullRequests, PullRequest{
			ID:                *pr.Number,
			Title:             *pr.Title,
			Description:       *pr.Body,
			Source:            &source{
				Branch: branch{
					Name: *pr.Head.Ref,
				},
			},
			Dest:              &dest{
				Branch: branch{
					Name: *pr.Base.Ref,
				},
			},
			CloseSourceBranch: true,
			State:             "OPEN",
			Reviewers:         reviewers,
		})
	}
	return pullRequests, nil
}

func (gh GitHub) DeclinePullRequest(username string, repoSlug string, pullRequestID int) error {
	httpClient := gh.authentication.AuthenticateWithToken()
	client := github.NewClient(httpClient)

	newState := "closed"
	pull := github.PullRequest{
		State: &newState,
	}
	pr, _, err := client.PullRequests.Edit(context.Background(), username, repoSlug, pullRequestID, &pull)

	if err != nil {
		log.Logger.Error("Error editing GitHub Pull Request")
		return nil
	}

	log.Logger.Info(fmt.Sprintf("Closed PR number %x", *pr.Number))

	return nil
}