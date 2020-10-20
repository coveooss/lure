package command_test

import (
	"regexp"
	"testing"

	"github.com/coveooss/lure/lib/lure/versionManager"

	"github.com/coveooss/lure/lib/lure/command"
	"github.com/coveooss/lure/lib/lure/project"
	managementsystem "github.com/coveooss/lure/lib/lure/repositorymanagementsystem"
)

type dummySourceControl struct {
}

func (d *dummySourceControl) Update(string) (string, error) {
	return "watev", nil
}

func (d *dummySourceControl) CommitsBetween(string, string) ([]string, error) {
	return []string{"watev"}, nil
}

func (d *dummySourceControl) Branch(string) (string, error) {
	return "watev", nil
}
func (d *dummySourceControl) SoftBranch(string) (string, error) {
	return "watev", nil
}
func (d *dummySourceControl) Push() (string, error) {
	return "watev", nil
}
func (d *dummySourceControl) WorkingPath() string {
	return "watev"
}
func (d *dummySourceControl) ActiveBranches() ([]string, error) {
	return []string{"watev"}, nil
}
func (d *dummySourceControl) CloseBranch(string) error {
	return nil
}
func (d *dummySourceControl) LocalPath() string {
	return "watev"
}
func (d *dummySourceControl) SanitizeBranchName(branchName string) string {
	reg, _ := regexp.Compile("[^a-zA-Z0-9_-]+")
	safe := reg.ReplaceAllString(branchName, "_")
	return safe
}
func (d *dummySourceControl) Commit(string) (string, error) {
	return "watev", nil
}

type dummyRepository struct {
	ExistingPrs           []managementsystem.PullRequest
	OpenPullRequestCalled bool
}

func (d *dummyRepository) CreatePullRequest(sourceBranch string, destBranch string, owner string, repo string, title string, description string, useDefaultReviewers bool) error {
	d.OpenPullRequestCalled = true
	return nil
}
func (d *dummyRepository) GetPullRequests(string, string, bool) ([]managementsystem.PullRequest, error) {
	return d.ExistingPrs, nil
}

func (d *dummyRepository) DeclinePullRequest(string, string, int) error {
	return nil
}

func (d *dummyRepository) GetURL() string {
	return ""
}

type dummyVersionControl struct {
	ModuleToReturn       []versionManager.ModuleVersion
	GetOutdatedError     error
	GetOutdatedWasCalled bool
}

func (d *dummyVersionControl) GetOutdated(path string) ([]versionManager.ModuleVersion, error) {
	d.GetOutdatedWasCalled = true
	return d.ModuleToReturn, d.GetOutdatedError
}

func (d *dummyVersionControl) UpdateDependency(path string, moduleVersion versionManager.ModuleVersion) (bool, error) {
	return true, nil
}

type dummyBranch struct {
	BranchName string
}

func (d *dummyBranch) GetName() string {
	return d.BranchName
}

func TestCheckForUpdatesJobCommandShouldNotOpenPRWhenPRAlreadyExists(t *testing.T) {

	skipPackageManageConfiguration := make(map[string]bool)
	skipPackageManageConfiguration["mvn"] = false

	mvn := &dummyVersionControl{}
	moduleToReturn := []versionManager.ModuleVersion{
		versionManager.ModuleVersion{
			ModuleUpdater: mvn,
			Module:        "yolo",
			Current:       "1.2.1",
			Latest:        "1.2.3",
			Wanted:        "1.2.3",
			Name:          "swag",
		},
	}
	mvn.ModuleToReturn = moduleToReturn

	existingPrs := []managementsystem.PullRequest{
		managementsystem.PullRequest{
			ID:     32,
			Title:  "Update maven dependency yolo.swag to version 1.2.3",
			Source: &dummyBranch{BranchName: "lure-swag-1_2_3-71334c00-b060-4830-86c0-c7077545712d"},
			Dest:   &dummyBranch{BranchName: "irrelevant"},
			State:  "OPEN",
		},
	}
	repository := &dummyRepository{ExistingPrs: existingPrs}

	useDefaultReviewers := false
	command.CheckForUpdatesJobCommand(project.Project{SkipPackageManager: skipPackageManageConfiguration, UseDefaultReviewers: &useDefaultReviewers}, &dummySourceControl{}, repository, make(map[string]string), mvn, &dummyVersionControl{})

	if repository.OpenPullRequestCalled {
		t.Log("Should not open a pull request")
		t.Fail()
	}

}

func TestCheckForUpdatesJobCommandShouldNotOpenPRWhenDecliendPRExists(t *testing.T) {

	skipPackageManageConfiguration := make(map[string]bool)
	skipPackageManageConfiguration["mvn"] = false

	mvn := &dummyVersionControl{}
	moduleToReturn := []versionManager.ModuleVersion{
		versionManager.ModuleVersion{
			ModuleUpdater: mvn,
			Module:        "yolo",
			Current:       "1.2.1",
			Latest:        "1.2.3",
			Wanted:        "1.2.3",
			Name:          "swag",
		},
	}
	mvn.ModuleToReturn = moduleToReturn

	existingPrs := []managementsystem.PullRequest{
		managementsystem.PullRequest{
			ID:     32,
			Title:  "Update maven dependency yolo.swag to version 1.2.3",
			Source: &dummyBranch{BranchName: "lure-swag-1_2_3-71334c00-b060-4830-86c0-c7077545712d"},
			Dest:   &dummyBranch{BranchName: "irrelevant"},
			State:  "DECLINED",
		},
	}
	repository := &dummyRepository{ExistingPrs: existingPrs}

	useDefaultReviewers := false
	command.CheckForUpdatesJobCommand(project.Project{SkipPackageManager: skipPackageManageConfiguration, UseDefaultReviewers: &useDefaultReviewers}, &dummySourceControl{}, repository, make(map[string]string), mvn, &dummyVersionControl{})

	if repository.OpenPullRequestCalled {
		t.Log("Should not open a pull request")
		t.Fail()
	}
}

func TestCheckForUpdatesJobCommandWithDecliendPRWithSamePrefix(t *testing.T) {

	skipPackageManageConfiguration := make(map[string]bool)
	skipPackageManageConfiguration["mvn"] = false

	mvn := &dummyVersionControl{}
	moduleToReturn := []versionManager.ModuleVersion{
		versionManager.ModuleVersion{
			ModuleUpdater: mvn,
			Module:        "yolo",
			Current:       "1.2.1",
			Latest:        "1.2.3",
			Wanted:        "1.2.3",
			Name:          "swag",
		},
	}
	mvn.ModuleToReturn = moduleToReturn

	existingPrs := []managementsystem.PullRequest{
		managementsystem.PullRequest{
			ID:     32,
			Title:  "Update maven dependency yolo.swag to version 1.2.3",
			Source: &dummyBranch{BranchName: "lure-swag-1_2_2-71334c00-b060-4830-86c0-c7077545712d"},
			Dest:   &dummyBranch{BranchName: "irrelevant"},
			State:  "DECLINED",
		},
	}
	repository := &dummyRepository{ExistingPrs: existingPrs}

	useDefaultReviewers := false
	command.CheckForUpdatesJobCommand(project.Project{SkipPackageManager: skipPackageManageConfiguration, UseDefaultReviewers: &useDefaultReviewers}, &dummySourceControl{}, repository, make(map[string]string), mvn, &dummyVersionControl{})

	if !repository.OpenPullRequestCalled {
		t.Log("Should have opened a pull request with the latest version")
		t.Fail()
	}
}

func TestCheckForUpdatesJobCommandShouldOpenAPRWhenNoPRWasOpenedForThis(t *testing.T) {

	skipPackageManageConfiguration := make(map[string]bool)
	skipPackageManageConfiguration["mvn"] = false

	mvn := &dummyVersionControl{}
	moduleToReturn := []versionManager.ModuleVersion{
		versionManager.ModuleVersion{
			ModuleUpdater: mvn,
			Module:        "AnOtherModule",
			Current:       "1.2.1",
			Latest:        "1.2.3",
			Wanted:        "1.2.3",
			Name:          "NoChill",
		},
	}
	mvn.ModuleToReturn = moduleToReturn

	existingPrs := []managementsystem.PullRequest{
		managementsystem.PullRequest{
			ID:     32,
			Title:  "Update maven dependency yolo.swag to version 1.2.3",
			Source: &dummyBranch{BranchName: "lure-swag-1_2_2-71334c00-b060-4830-86c0-c7077545712d"},
			Dest:   &dummyBranch{BranchName: "irrelevant"},
			State:  "DECLINED",
		},
	}
	repository := &dummyRepository{ExistingPrs: existingPrs}

	useDefaultReviewers := false
	command.CheckForUpdatesJobCommand(project.Project{SkipPackageManager: skipPackageManageConfiguration, UseDefaultReviewers: &useDefaultReviewers}, &dummySourceControl{}, repository, make(map[string]string), mvn, &dummyVersionControl{})

	if !repository.OpenPullRequestCalled {
		t.Log("Should have opened a pull request with the latest version")
		t.Fail()
	}
}

func TestNpmFailureShouldNotPreventMvnUpdate(t *testing.T) {

	skipPackageManageConfiguration := make(map[string]bool)
	skipPackageManageConfiguration["mvn"] = false
	skipPackageManageConfiguration["npm"] = false

	npm := &dummyVersionControl{}
	mvn := &dummyVersionControl{}

	repository := &dummyRepository{}

	useDefaultReviewers := false
	command.CheckForUpdatesJobCommand(project.Project{SkipPackageManager: skipPackageManageConfiguration, UseDefaultReviewers: &useDefaultReviewers}, &dummySourceControl{}, repository, make(map[string]string), mvn, npm)

	if !mvn.GetOutdatedWasCalled {
		t.Log("Should have called GetOutdated for Mvn")
		t.Fail()
	}
}
