package main

import (
	"github.com/spf13/cobra"
	"github.com/water-hole/poc-retablir/cmd/apply"
	"github.com/water-hole/poc-retablir/cmd/export"
	"github.com/water-hole/poc-retablir/cmd/transform"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os"
)

func main() {
	//flags := pflag.NewFagSet("kubectl-ns", pflag.ExitOnError)
	//pflag.CommandLine = flags

	//root := extract.NewExtractorOptions(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})

	//if err := root.Execute(); err != nil {
	//	os.Exit(1)
	//}
	root := cobra.Command{
		Use: "retablir",
	}
	root.AddCommand(export.NewExportCommand(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}))
	root.AddCommand(transform.NewTransformCommand())
	root.AddCommand(apply.NewApplyCommand())
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
