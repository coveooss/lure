# lure

[![Build Status](https://travis-ci.org/coveo/lure.svg)](https://travis-ci.org/coveo/lure)
[![Go Report Card](https://goreportcard.com/badge/github.com/coveo/lure)](https://goreportcard.com/report/github.com/coveo/lure)

Update your dependencies, with hooks, for developers.

## Setup your repository

First create a `lure.config` in the repository you want to keep your dependencies up-to-date.

The file should look like:

```
{
    "projects": [{
            "vcs": "hg",
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
- `hg` for mercurial
- `git` for git

The possible commands are:
- `updateDependencies`
- `synchronizedBranches`

Other:
- `owner`: https ://bitbucket.org/**owner**/name
- `name`: https ://bitbucket.org/owner/**name**
- `skipPackageManager` (Optional):  Allows to explicitely skip a package manager update. Allowed keys are: `npm` and `mvn`.
- `useDefaultReviewers` (Optional): True by default, allows NOT using the default reviewer list on pull requests.

## Setup your CI

eg, in jenkins:

```env
git config --global user.email "Youmail@example.com"
git config --global user.name "jenkins"

wget https://github.com/coveo/lure/releases/latest/download/lure-linux-amd64 -O lure
chmod +x lure
./lure -auth env -config ${WORKSPACE}/lure.config

```

You need bitbucket api-key and api-secret, see, the [bitbucket documentation](https://confluence.atlassian.com/bitbucket/oauth-on-bitbucket-cloud-238027431.html#OAuthonBitbucketCloud-OAuth2.0) for OAuth setup.

Environment variables:
- `IGNORE_DECLINED_PR=1` Will ignore declined PR when looking if the PR exists
- `BITBUCKET_CLIENT_ID` the bitbucket OAuth **Key** previously created
- `BITBUCKET_CLIENT_SECRET` the bitbucket OAuth **Secret** previously created
- `DRY_RUN` won't create a PR

Custom parameter:
- `-verbose` Will print additionnal logs that could be helpful for debugging

## Develop

### GO environment setup

if you're an old time go user you already know what to do

```sh
mkdir -p $HOME/go/
```

### Create bitbucket APP
In your bitbucket app, set the callback url to `http://localhost:9090/callback`

### Project setup

```sh
go get github.com/coveo/lure/
cd $GOPATH/src/github.com/coveo/lure
# or $HOME/go/src/coveo/lure if you don't have a $GOPATH set up which is perfectly fine
go get ./...
go run lure.go
```

Build:

```sh
go build lure.go
```

For more information about building, you can check the BUILD.md.
