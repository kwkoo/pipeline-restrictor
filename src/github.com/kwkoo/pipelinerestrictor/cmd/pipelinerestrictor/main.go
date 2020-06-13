package main

import (
	"github.com/kwkoo/pipelinerestrictor"
	"github.com/openshift/generic-admission-server/pkg/cmd"
)

func main() {
	cmd.RunAdmissionServer(&pipelinerestrictor.AdmissionHook{})
}
