package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/bitbucket"

	"github.com/coveo/lure/lib/lure"
)

var (
	mode     = flag.String("auth", "", "one of [oauth, env]")
	confFile = flag.String("config", "", "path to config file")

	bitBucketOAuthConfig = oauth2.Config{
		ClientID:     os.Getenv("BITBUCKET_CLIENT_ID"),
		ClientSecret: os.Getenv("BITBUCKET_CLIENT_SECRET"),
		Endpoint:     bitbucket.Endpoint,
	}
)

type CommandFunc func(auth lure.Authentication, project lure.Project, args map[string]string) error
type Main func(config *lure.LureConfig)

func main() {
	flag.Parse()

	config, err := loadConfig(*confFile)
	if err != nil {
		log.Printf("Error Loading Config: %s\n", err)
		os.Exit(1)
	}
	log.Printf("Config: %s\n", config)

	var mainFunc Main = nil
	switch *mode {
	case "oauth":
		log.Println("Using OAuth Authentication")
		mainWithOAuth(config)
	case "env":
		log.Println("Using Environment Authentication")
		mainWithEnvironmentAuth(config)
	default:
		log.Printf("Invalid auth mode: %s", *mode)
		flag.PrintDefaults()
		os.Exit(1)
	}

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

	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, bitBucketOAuthConfig.AuthCodeURL(""), http.StatusFound)
	})

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.FormValue("error") != "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("There was an error authenticating with google"))
			return
		}

		if r.FormValue("code") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Code is not present"))
			return
		}

		token, err := bitBucketOAuthConfig.Exchange(oauth2.NoContext, r.FormValue("code"))
		if err != nil || token == nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("There was an error with the token exchange: error:'%s', token: '%s' ", err.Error(), token)))
			return
		}

		log.Println("Token is", token)

		w.WriteHeader(http.StatusFound)
		w.Write([]byte("Linking with Bitbucket worked - get out and wait for an update"))

		go (func() {
			auth := lure.TokenAuth{token.AccessToken}

			runMain(config, auth)
			os.Exit(0)
		})()
	})

	log.Println("Open that page: http://localhost:9090/login")
	http.ListenAndServe(":9090", mux)
}

func loadConfig(filePath string) (*lure.LureConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	lureConfig := &lure.LureConfig{}
	if err := json.NewDecoder(file).Decode(lureConfig); err != nil {
		return nil, err
	}
	return lureConfig, nil
}
