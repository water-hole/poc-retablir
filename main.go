package main

import (
	"github.com/spf13/cobra"
<<<<<<< HEAD
	"github.com/water-hole/poc-retablir/cmd/export"
=======
	"github.com/water-hole/poc-retablir/cmd/extract"
	"github.com/water-hole/poc-retablir/cmd/transform"
>>>>>>> bde0c1b... adding transformation CLI. Will take the output from export.
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
<<<<<<< HEAD
	root.AddCommand(export.NewExportCommand(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}))
=======
	root.AddCommand(extract.NewExtractCommand(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}))
	root.AddCommand(transform.NewTransformCommand())
>>>>>>> bde0c1b... adding transformation CLI. Will take the output from export.
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
