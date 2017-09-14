package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/bitbucket"

	"github.com/coveo/lure/lib/lure"
)

func main() {
	mode := flag.String("auth", "", "one of [oauth, env]")
	confFile := flag.String("config", "", "path to config file")
	flag.Parse()

	var mainFunc Main = nil
	switch *mode {
	case "oauth":
		log.Println("Using OAuth Authentication")
		mainFunc = mainWithOAuth
	case "env":
		log.Println("Using Environment Authentication")
		mainFunc = mainWithEnvironmentAuth
	default:
		fmt.Printf("Invalid auth: %s", *mode)
		os.Exit(1)
	}

	config, err := loadConfig(*confFile)
	if err != nil {
		fmt.Printf("Error Loading Config: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Config: %s\n", config)

	mainFunc(config)
}

func getCommand(commandName string) CommandFunc {
	var commandFunc CommandFunc = nil

	switch commandName {
	case "updateDependencies":
		commandFunc = lure.CheckForUpdatesJobCommand
	case "synchronizedBranches":
		commandFunc = lure.SynchronizedBranchesCommand
	}

	return commandFunc
}

func runMain(config *lure.LureConfig, auth lure.Authentication) {
	for _, project := range config.Projects {
		log.Println(fmt.Sprintf("Project: %s/%s", project.Owner, project.Name))

		for _, command := range project.Commands {
			log.Println(fmt.Sprintf("\tCommand: %s", command.Name))
			commandFunc := getCommand(command.Name)

			if commandFunc == nil {
				log.Println(fmt.Sprintf("\tSkipping invalid command: %s", command.Name))
			} else {
				if err := commandFunc(auth, project, command.Args); err != nil {
					log.Println(fmt.Sprintf("\tCommand failed: %s", err))
				}
			}
		}
	}
}

func mainWithEnvironmentAuth(config *lure.LureConfig) {

	auth := lure.UserPassAuth{
		Username: os.Getenv("BITBUCKET_USERNAME"),
		Password: os.Getenv("BITBUCKET_PASSWORD"),
	}

	runMain(config, auth)
}

func mainWithOAuth(config *lure.LureConfig) {

	r := gin.Default()
	r.GET("/login", func(c *gin.Context) {
		fmt.Println("plz")
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

		fmt.Println(token)

		if token == nil {
			respondWithError(http.StatusInternalServerError, "There was an error with the token exchange, no error, but no token either", c)
			return
		}

		c.String(http.StatusFound, "Linking with Bitbucket worked - get out and wait for an update")

		go (func() {
			auth := lure.TokenAuth{token.AccessToken}

			runMain(config, auth)
			os.Exit(0)
		})()
	})
	fmt.Println("--------GO THERE ", bitBucketOAuthConfig.AuthCodeURL(""))
	go r.Run(":9090")

	lure.Execute("", "open", "http://localhost:9090/login")
	for {
		time.Sleep(5000 * time.Second)
	}
}

var (
	bitBucketOAuthConfig = oauth2.Config{
		ClientID:     os.Getenv("BITBUCKET_CLIENT_ID"),
		ClientSecret: os.Getenv("BITBUCKET_CLIENT_SECRET"),
		Endpoint:     bitbucket.Endpoint,
	}
)

func loadConfig(filePath string) (*lure.LureConfig, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	lureConfig := &lure.LureConfig{}
	if err := json.NewDecoder(bytes.NewReader(data)).Decode(lureConfig); err != nil {
		return nil, err
	}
	return lureConfig, nil
}

func respondWithError(code int, message string, c *gin.Context) {
	resp := map[string]string{"error": message}

	c.JSON(code, resp)
}

type CommandFunc func(auth lure.Authentication, project lure.Project, args map[string]string) error
type Main func(config *lure.LureConfig)
