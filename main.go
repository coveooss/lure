package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/k0kubun/pp"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/bitbucket"
)

var (
	project = Project{
		Remote:        "bitbucket.org/pastjean/dummy",
		DefaultBranch: "default",
	}
	bitBucketOAuthConfig = oauth2.Config{
		ClientID:     os.Getenv("BITBUCKET_CLIENT_ID"),
		ClientSecret: os.Getenv("BITBUCKET_CLIENT_SECRET"),
		Endpoint:     bitbucket.Endpoint,
	}
)

func respondWithError(code int, message string, c *gin.Context) {
	resp := map[string]string{"error": message}

	c.JSON(code, resp)
}
func main() {
	r := gin.Default()
	r.GET("/login", func(c *gin.Context) {
		pp.Println("plz")
		c.Redirect(302, bitBucketOAuthConfig.AuthCodeURL(""))
	})

	r.GET("/callback", func(c *gin.Context) {
		if len(c.Request.FormValue("error")) != 0 {
			respondWithError(http.StatusUnauthorized, "There was an error authenticating with google", c)
			return
		}
		if len(c.Request.FormValue("code")) == 0 {
			respondWithError(http.StatusUnauthorized, "Code is not present", c)
			return
		}

		token, err := bitBucketOAuthConfig.Exchange(oauth2.NoContext, c.Request.FormValue("code"))
		if err != nil {
			respondWithError(http.StatusInternalServerError, "There was an error with the token exchange"+err.Error(), c)
			return
		}

		project.Token = token

		pp.Println(token)
		c.String(http.StatusFound, "Linking with Bitbucket worked - get out and wait for an update")
	})
	fmt.Println("--------GO THERE ", bitBucketOAuthConfig.AuthCodeURL(""))
	go r.Run(":9090")
	go checkForUpdatesJob([]*Project{&project})

	for {
		time.Sleep(1000 * time.Second)
	}
}
