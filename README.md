# lure

web hooks, for devs.

## env.
`IGNORE_DECLINED_PR=1` Will ignore declined PR when looking if the PR existss

See [bitbucket documentation](https://confluence.atlassian.com/bitbucket/oauth-on-bitbucket-cloud-238027431.html#OAuthonBitbucketCloud-OAuth2.0) for OAuth setup.

`BITBUCKET_CLIENT_ID` the bitbucket OAuth **Key**

`BITBUCKET_CLIENT_SECRET` the bitbucket OAuth **Secret**

`BITBUCKET_REPO_NAME` https ://bitbucket.org/owner/**name**

`BITBUCKET_REPO_OWNER` https ://bitbucket.org/**owner**/name

`DRY_RUN` won't create a PR

`GOPATH`= project root

## dependencies.

`go get -v ./...`

## test.

```sh
go run lure.go
```

## jenkins

```sh
wget https://github.com/coveo/Lure/releases/download/v1.0/lure-linux-amd64
chmod +x lure-linux-amd64
./lure-linux-amd64 -auth env -config ${WORKSPACE}/lure.config
```
