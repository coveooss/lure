package os

import (
	"bytes"
	"os/exec"

	"github.com/coveooss/lure/lib/lure/log"
)

func Execute(pwd string, command string, params ...string) (string, error) {
	log.Logger.Tracef("%s %q", command, params)

	cmd := exec.Command(command, params...)
	cmd.Dir = pwd

	var buff bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &buff
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Logger.Error(stderr.String())
		return "", err
	}

	out := buff.String()

	log.Logger.Tracef("\t%s\n", out)

	return out, nil
}
