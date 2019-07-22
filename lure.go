package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/coveooss/lure/lib/lure/command"
	"github.com/coveooss/lure/lib/lure/log"
	"github.com/coveooss/lure/lib/lure/project"
	"github.com/coveooss/lure/lib/lure/provider"
	"github.com/coveooss/lure/lib/lure/vcs"
	"github.com/sirupsen/logrus"
	"github.com/vsekhar/govtil/guid"

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

func main() {
	flag.Parse()

	if *verbose {
		log.Logger.Info("Log level set to verbose")
		log.Logger.SetLevel(logrus.TraceLevel)
	} else {
		log.Logger.SetLevel(logrus.InfoLevel)
	}

	log.Logger.SetOutput(os.Stdout)
	log.Logger.SetReportCaller(true)

	config, err := loadConfig(*confFile)
	if err != nil {
		log.Logger.Error(fmt.Sprintf("Error Loading Config with path '%s': %s\n", *confFile, err))
		os.Exit(1)
	}

	if os.Getenv("DRY_RUN") == "1" {
		log.Logger.Info("Running in DryRun mode, not doing the pull request nor pushing the changes")
	}

	switch *mode {
	case "oauth":
		log.Logger.Info("Using OAuth Authentication")
		mainWithOAuth(config)
	case "env":
		log.Logger.Info("Using Environment Authentication")
		mainWithEnvironmentAuth(config)
	default:
		log.Logger.Errorf("Invalid auth mode: %s", *mode)
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func runMain(config *project.LureConfig, auth vcs.Authentication) {
	for _, projectConfig := range config.Projects {
		log.Logger.Info(fmt.Sprintf("Project: %s/%s", projectConfig.Owner, projectConfig.Name))

		project.InitProjectDefaultValues(&projectConfig)

		repoGUID, err := guid.V4()

		if err != nil {
			log.Logger.Fatalf("\"Could not generate guid\" %s", err)
		}
		localDestination := "/tmp/" + repoGUID.String()

		provider := provider.New(auth, projectConfig)

		var sourceControl vcs.SourceControl
		switch projectConfig.Vcs {
		case vcs.Hg:
			sourceControl, err = vcs.NewHg(auth, provider.URL, localDestination, projectConfig.GetDefaultBranch(), projectConfig.GetDefaultBranch(), projectConfig.GetBasePath())
		case vcs.Git:
			sourceControl, err = vcs.NewGit(auth, provider.URL, localDestination, projectConfig.GetBasePath())
		default:
			//repo = nil
			err = fmt.Errorf("Unknown VCS '%s' - must be one of %s, %s", projectConfig.Vcs, vcs.Git, vcs.Hg)
		}

		sourceControl.Clone()
		

		for _, cmd := range projectConfig.Commands {
			log.Logger.Info(fmt.Sprintf("Command: %s", cmd.Name))
			var err error
			switch cmd.Name {
			case "updateDependencies":
				err = command.CheckForUpdatesJobCommand(projectConfig, sourceControl, provider, cmd.Args)
			case "synchronizedBranches":
				err = command.SynchronizedBranchesCommand(projectConfig, sourceControl, provider, cmd.Args)
			default:
				log.Logger.Info(fmt.Sprintf("\tSkipping invalid command: %s", cmd.Name))
			}

			if err != nil {
				log.Logger.Error(fmt.Sprintf("Command failed: %s", err))
				os.Exit(1)
			}
		}
	}
}

func mainWithEnvironmentAuth(config *project.LureConfig) {
	auth := vcs.UserPassAuth{
		Username: os.Getenv("BITBUCKET_USERNAME"),
		Password: os.Getenv("BITBUCKET_PASSWORD"),
	}

	runMain(config, auth)
}

func mainWithOAuth(config *project.LureConfig) {
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

		log.Logger.Println("Token is", token)

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
			auth := vcs.TokenAuth{token.AccessToken}

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
		log.Logger.Info("Open that page: " + url)
	}

	err := http.ListenAndServe(":"+port, mux)
	if err != nil {
		log.Logger.Error(fmt.Sprintf("Error starting the webserver: %s", err))
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

func loadConfig(filePath string) (*project.LureConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	lureConfig := &project.LureConfig{}
	if err := json.NewDecoder(file).Decode(lureConfig); err != nil {
		return nil, err
	}
	configJson, _ := json.Marshal(lureConfig)
	log.Logger.Trace("Config:", string(configJson))
	return lureConfig, nil
}
