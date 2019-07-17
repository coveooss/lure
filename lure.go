package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/coveooss/lure/lib/lure"
	"github.com/sirupsen/logrus"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/bitbucket"
)

var (
	mode     = flag.String("auth", "", "one of [oauth, env]")
	confFile = flag.String("config", "", "path to config file")
	verbose  = flag.Bool("verbose", false, "set to true to show more logs")

	bitBucketOAuthConfig = oauth2.Config{
		ClientID:     os.Getenv("BITBUCKET_CLIENT_ID"),
		ClientSecret: os.Getenv("BITBUCKET_CLIENT_SECRET"),
		Endpoint:     bitbucket.Endpoint,
	}
)

type CommandFunc func(auth lure.Authentication, project lure.Project, args map[string]string) error

func main() {
	flag.Parse()

	if *verbose {
		lure.Logger.Info("Log level set to verbose")
		lure.Logger.SetLevel(logrus.TraceLevel)
	} else {
		lure.Logger.SetLevel(logrus.InfoLevel)
	}

	lure.Logger.SetOutput(os.Stdout)

	config, err := loadConfig(*confFile)
	if err != nil {
		lure.Logger.Error(fmt.Sprintf("Error Loading Config with path '%s': %s\n", *confFile, err))
		os.Exit(1)
	}
	if os.Getenv("DRY_RUN") == "1" {
		lure.Logger.Info("Running in DryRun mode, not doing the pull request nor pushing the changes")
	}

	switch *mode {
	case "oauth":
		lure.Logger.Info("Using OAuth Authentication")
		mainWithOAuth(config)
	case "env":
		lure.Logger.Info("Using Environment Authentication")
		mainWithEnvironmentAuth(config)
	default:
		lure.Logger.Error("Invalid auth mode: %s", *mode)
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func getCommand(commandName string) CommandFunc {
	switch commandName {
	case "updateDependencies":
		return lure.CheckForUpdatesJobCommand
	case "synchronizedBranches":
		return lure.SynchronizedBranchesCommand
	}

	return nil
}

func runMain(config *lure.LureConfig, auth lure.Authentication) {
	for _, project := range config.Projects {
		lure.Logger.Info(fmt.Sprintf("Project: %s/%s", project.Owner, project.Name))

		lure.InitProjectDefaultValues(&project)

		for _, command := range project.Commands {
			lure.Logger.Info(fmt.Sprintf("Command: %s", command.Name))
			commandFunc := getCommand(command.Name)

			if commandFunc == nil {
				lure.Logger.Info(fmt.Sprintf("\tSkipping invalid command: %s", command.Name))
			} else {
				if err := commandFunc(auth, project, command.Args); err != nil {
					lure.Logger.Error(fmt.Sprintf("Command failed: %s", err))
					os.Exit(1)
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

		lure.Logger.Println("Token is", token)

		w.WriteHeader(http.StatusFound)
		w.Write([]byte(`<html><body>Linking with Bitbucket worked - get out and wait for an update<script type="text/javascript">
		function closeWindow() {
		   setTimeout(function() {
		   window.close();
		   }, 3000);
		   }	   
		   window.onload = closeWindow();
		   </script></body></html>`))

		go (func() {
			auth := lure.TokenAuth{token.AccessToken}

			runMain(config, auth)
			os.Exit(0)
		})()
	})

	port := os.Getenv("LURE_WEBSERVER_PORT")
	if len(port) == 0 {
		port = "9090"
	}

	var url = "http://localhost:" + port + "/login"
	if os.Getenv("LURE_AUTO_OPEN_AUTH_PAGE") == "1" {
		open(url)
	} else {
		lure.Logger.Info("Open that page: " + url)
	}

	err := http.ListenAndServe(":"+port, mux)
	if err != nil {
		lure.Logger.Error(fmt.Sprintf("Error starting the webserver: %s", err))
	}
}

func open(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
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
	configJson, _ := json.Marshal(lureConfig)
	lure.Logger.Println("Config:", string(configJson))
	return lureConfig, nil
}
