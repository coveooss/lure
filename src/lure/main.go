package main

import (
	"fmt"
	"net/http"
	"os"
	"io/ioutil"
	"time"
	"bytes"

	"github.com/gin-gonic/gin"
	"github.com/k0kubun/pp"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/bitbucket"
	"encoding/json"
)

var (
	bitBucketOAuthConfig = oauth2.Config{
		ClientID:     os.Getenv("BITBUCKET_CLIENT_ID"),
		ClientSecret: os.Getenv("BITBUCKET_CLIENT_SECRET"),
		Endpoint:     bitbucket.Endpoint,
	}
)

func loadConfig() (*LureConfig, error) {
	data, err := ioutil.ReadFile("lure.config")
	if err != nil {
		return nil, err
	}

	var lureConfig *LureConfig = &LureConfig{}
	if err := json.NewDecoder(bytes.NewReader(data)).Decode(lureConfig); err != nil {
		return nil, err
	}
	return lureConfig, nil
}

func respondWithError(code int, message string, c *gin.Context) {
	resp := map[string]string{"error": message}

	c.JSON(code, resp)
}
func main() {

	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Error Loading Config: %s\n", err)
		return
	}
	fmt.Printf("Config: %s\n", config)
	projects := config.Projects

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

		pp.Println(token)

		c.String(http.StatusFound, "Linking with Bitbucket worked - get out and wait for an update")

		go (func() {
			checkForUpdatesJob(token, projects)
			//checkForBranchDifferencesJob(token, projects, "staging", "default")
			os.Exit(0)
		})()
	})
	fmt.Println("--------GO THERE ", bitBucketOAuthConfig.AuthCodeURL(""))
	go r.Run(":9090")

	execute("", "open", "http://localhost:9090/login")
	for {
		time.Sleep(5000 * time.Second)
	}
}
