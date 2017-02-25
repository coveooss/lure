#!/bin/bash

export GOPATH=$(pwd)
export GOBIN=$(pwd)/bin

go run src/lure/*.go
