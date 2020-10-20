# lure

[![Build Status](https://travis-ci.org/coveooss/lure.svg)](https://travis-ci.org/coveooss/lure)
[![Go Report Card](https://goreportcard.com/badge/github.com/coveooss/lure)](https://goreportcard.com/report/github.com/coveooss/lure)

Update your dependencies, with hooks, for developers.

## Setup your repository

First create a `lure.config` in the repository you want to keep your dependencies up-to-date.

The file should look like:

```
{
    "projects": [{
            "vcs": "hg",
            "host": "bitbucket",
            "owner": "dreisch",
            "name": "catfeederhg",
            "defaultBranch": "default",
            "branchPrefix": "lure-",
            "useDefaultReviewers": false,
            "skipPackageManager": {
                "mvn": true,
                "npm": false
            },
            "commands": [{
                "name": "updateDependencies",
                "args": {
                    "commitMessage": "Update {{.module}} to {{.version}}\nMYJIRA-1234",
                    "pullRequestDescription": "{{.module}} version {{.version}} is now available! Please update.\nMYJIRA-1234"
                }
            }]
        },
        {
            "vcs": "git",
            "host": "github",
            "owner": "dreisch",
            "name": "catfeedergit",
            "defaultBranch": "master",
            "branchPrefix": "lure-",
            "commands": [{
                "name": "synchronizedBranches",
                "args": {
                    "from": "staging",
                    "to": "develop"
                }
            }]
        }
    ]
}
```

Possible vcs are:
- `git` for git
- `hg` for mercurial

Possible hosts are `github` and `bitbucket`. For now, `bitbucket` is the default.

The possible commands are:
- `updateDependencies`
- `synchronizedBranches`

Other:
- `owner`: https ://bitbucket.org/**owner**/name or https ://github.com/**owner**/name
- `name`: https ://bitbucket.org/owner/**name** or https ://github.com/owner/**name**
- `skipPackageManager` (Optional):  Allows to explicitly skip a package manager update. Allowed keys are: `npm` and `mvn`.
- `useDefaultReviewers` (Optional): True by default, allows NOT using the default reviewer list on pull requests.

## Setup your CI

eg, in jenkins:

```env
git config --global user.email "Youmail@example.com"
git config --global user.name "jenkins"

wget https://github.com/coveooss/lure/releases/latest/download/lure-linux-amd64 -O lure
chmod +x lure
./lure -auth env -config ${WORKSPACE}/lure.config

```

Environment variables:

- `IGNORE_DECLINED_PR=1` Will ignore declined PR when looking if the PR exists
- `LURE_AUTO_OPEN_AUTH_PAGE` automaticaly open the browser when using OAuth
- `DRY_RUN` won't create a PR

With Bitbucket:
You need bitbucket api-key and api-secret, see, the [bitbucket documentation](https://confluence.atlassian.com/bitbucket/oauth-on-bitbucket-cloud-238027431.html#OAuthonBitbucketCloud-OAuth2.0) for OAuth setup.

- `BITBUCKET_CLIENT_ID` the bitbucket OAuth **Key** previously created
- `BITBUCKET_CLIENT_SECRET` the bitbucket OAuth **Secret** previously created

With GitHub:
You can provide an access token from a GitHub App: `GITHUB_ACCESS_TOKEN` 

Another option is to provide a username and password. When using a Personal Access Token, for example:
- `GITHUB_USERNAME`
- `GITHUB_PASSWORD`

Custom parameter:
- `-verbose` Will print additional logs that could be helpful for debugging

## Develop

### GO environment setup

if you're an old time go user you already know what to do

```sh
mkdir -p $HOME/go/
```

### Create bitbucket APP
In your bitbucket app, set the callback url to `http://localhost:9090/callback`

### Project setup

Build:

```sh
go build lure.go
```

For more information about building, you can check the BUILD.md.
