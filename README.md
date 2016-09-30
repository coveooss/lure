# lure

web hooks, for devs.

## env.
`IGNORE_DELINED_PR=1` Will ignore declined PR when looking if the PR existss

See [bitbucket documentation](https://confluence.atlassian.com/bitbucket/oauth-on-bitbucket-cloud-238027431.html#OAuthonBitbucketCloud-OAuth2.0) for OAuth setup.

`BITBUCKET_CLIENT_ID` the bitbucket OAuth **Key**

`BITBUCKET_CLIENT_SECRET` the bitbucket OAuth **Secret**

`BITBUCKET_REPO_NAME` https ://bitbucket.org/owner/**name**

`BITBUCKET_REPO_OWNER` https ://bitbucket.org/**owner**/name

## test.

```sh
cat event.json | go run functions/hello-go/main.go
```
