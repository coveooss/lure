#!/bin/bash

export GOPATH=$(pwd)
export GOBIN=$GOPATH/bin

go run *.go
