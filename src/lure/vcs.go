package main

import (
)

type Authentication interface {}

type TokenAuth struct {
	token string
}

type UserPassAuth struct {
	username string
	password string
}

type Repo interface {
	LocalPath()  string
	RemotePath() string

	Cmd(args ...string) (string, error)
	Update(rev string) (string, error)
	Branch(branchname string) (string, error)
	Commit(message string) (string, error)
	Push() (string, error)

	LogCommitsBetween(baseRev string, secondRev string) ([]string, error)
}

const (
	Git = "git"
	Hg = "hg"
)
