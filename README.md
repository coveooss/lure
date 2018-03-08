# lure

[![Build Status](https://travis-ci.org/coveo/lure.svg)](https://travis-ci.org/coveo/lure)
[![Go Report Card](https://goreportcard.com/badge/github.com/coveo/lure)](https://goreportcard.com/report/github.com/coveo/lure)

Update your dependencies, with hooks, for developers.

## Run

eg, in jenkins:

```sh
wget https://github.com/coveo/Lure/releases/download/v1.0/lure-linux-amd64
chmod +x lure-linux-amd64
./lure-linux-amd64 -auth env -config <yourconfig>
```

You need bitbucket api-key and api-secret, see, the [bitbucket documentation](https://confluence.atlassian.com/bitbucket/oauth-on-bitbucket-cloud-238027431.html#OAuthonBitbucketCloud-OAuth2.0) for OAuth setup.

- `IGNORE_DECLINED_PR=1` Will ignore declined PR when looking if the PR exists
- `BITBUCKET_CLIENT_ID` the bitbucket OAuth **Key** previously created
- `BITBUCKET_CLIENT_SECRET` the bitbucket OAuth **Secret** previously created
- `BITBUCKET_REPO_NAME` https ://bitbucket.org/owner/**name**
- `BITBUCKET_REPO_OWNER` https ://bitbucket.org/**owner**/name
- `DRY_RUN` won't create a PR

## Develop

### GO environment setup

if you're an old time go user you already know what to do

```sh
mkdir -p $HOME/go/
```

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