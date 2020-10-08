package versionManager

type UpdateFunc func() error

// Allow the module to be updated
type ModuleUpdater interface {
	UpdateDependency(path string, moduleVersion ModuleVersion) (bool, error)
}

type ModuleVersion struct {
	Type          string
	Module        string
	Current       string
	Latest        string
	Wanted        string
	Name          string
	ModuleUpdater ModuleUpdater
}
