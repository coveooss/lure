package main

import (

)

type UpdateFunc func() error

type moduleVersion struct {
	Type    string
	Module  string
	Current string
	Latest  string
	Wanted  string
}
