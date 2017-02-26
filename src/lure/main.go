package main

import (
	"fmt"
	"net/http"
	"os"
	"log"
	"io/ioutil"
	"time"
	"bytes"
	"flag"

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

type CommandFunc func(auth Authentication, project Project, args map[string]string) error
type Main func(config *LureConfig)

func main() {
	mode := flag.String("auth", "", "one of [oauth, env]")

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
		fmt.Printf("Invalid auth: %s", mode)
		os.Exit(1)
	}

	config, err := loadConfig()
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
		commandFunc = checkForUpdatesJobCommand
	case "synchronizedBranches":
		commandFunc = synchronizedBranchesCommand
	}

	return commandFunc
}

func runMain(config *LureConfig, auth Authentication) {
	for _, project := range config.Projects {
		log.Println(fmt.Sprintf("Project: %s/%s", project.Owner , project.Name))

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

func mainWithEnvironmentAuth(config *LureConfig) {

	auth := UserPassAuth{
		username: os.Getenv("BITBUCKET_USERNAME"),
		password: os.Getenv("BITBUCKET_PASSWORD"),
	}

	runMain(config, auth)
}

func mainWithOAuth(config *LureConfig) {

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

		if token == nil {
			respondWithError(http.StatusInternalServerError, "There was an error with the token exchange, no error, but no token either", c)
			return
		}

		c.String(http.StatusFound, "Linking with Bitbucket worked - get out and wait for an update")

		go (func() {
			auth := TokenAuth{token.AccessToken}

			runMain(config, auth)
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
