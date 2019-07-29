package versionManager

type UpdateFunc func() error

type ModuleVersion struct {
	Type    string
	Module  string
	Current string
	Latest  string
	Wanted  string
	Name    string
}
