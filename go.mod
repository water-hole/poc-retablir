module github.com/water-hole/poc-retablir

go 1.16

replace github.com/konveyor/transformations => github.com/water-hole/transformations v0.0.0-20210525213018-deb680135f38

require (
	github.com/evanphx/json-patch v4.9.0+incompatible
	github.com/konveyor/transformations v1.0.0
	github.com/spf13/cobra v1.1.1
	k8s.io/apimachinery v0.20.4
	k8s.io/cli-runtime v0.20.4
	k8s.io/client-go v0.20.4
	sigs.k8s.io/kustomize/kyaml v0.10.15
	sigs.k8s.io/yaml v1.2.0
)
