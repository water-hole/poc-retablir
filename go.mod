module github.com/water-hole/poc-retablir

go 1.16

replace github.com/konveyor/transformations => /home/shurley/repos/konveyor/transformations

require (
	github.com/konveyor/transformations v1.0.0
	github.com/spf13/cobra v1.1.1
	k8s.io/apimachinery v0.20.4
	k8s.io/cli-runtime v0.20.4
	k8s.io/client-go v0.20.4
	sigs.k8s.io/yaml v1.2.0
)
