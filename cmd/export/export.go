package export

// TODO: alpatel this will eventually make it into a library, I think

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	errorsutil "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd/api"
)

type ExportOptions struct {
	configFlags *genericclioptions.ConfigFlags

	ExportDir string
	Context   string
	Namespace string
	genericclioptions.IOStreams
}

func (o *ExportOptions) Complete(c *cobra.Command, args []string) error {
	// TODO: @alpatel
	return nil
}

func (o *ExportOptions) Validate() error {
	// TODO: @alpatel
	return nil
}

func (o *ExportOptions) Run() error {
	return o.run()
}

func NewExportCommand(streams genericclioptions.IOStreams) *cobra.Command {
	o := &ExportOptions{
		configFlags: genericclioptions.NewConfigFlags(true),

		IOStreams: streams,
	}
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export the namespace resources in an output directory",
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	addFlagsForOptions(o, cmd)

	return cmd
}

func addFlagsForOptions(o *ExportOptions, cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.ExportDir, "export-dir", "export", "The path where files are to be exported")
	cmd.Flags().StringVar(&o.Context, "context", "", "The kube context, if empty it will use the current context")
	cmd.Flags().StringVar(&o.Namespace, "namespace", "", "The kube namespace to export. If --context is set it will try to get the namespace from the context and that will take precedence")
}

func (o *ExportOptions) run() error {
	config := o.configFlags.ToRawKubeConfigLoader()
	rawConfig, err := config.RawConfig()
	if err != nil {
		fmt.Printf("error in generating raw config")
		os.Exit(1)
	}
	if o.Context == "" {
		o.Context = rawConfig.CurrentContext
	}

	if o.Context == "" {
		fmt.Printf("current kubecontext is empty and not kubecontext is specified")
		os.Exit(1)
	}

	var currentContext *api.Context
	contextName := ""

	for name, ctx := range rawConfig.Contexts {
		if name == o.Context {
			currentContext = ctx
			contextName = name
		}
	}

	if currentContext == nil {
		fmt.Printf("currentContext is nil")
		os.Exit(1)
	}

	if len(currentContext.Namespace) > 0 {
		o.Namespace = currentContext.Namespace
	}

	if o.Namespace == "" {
		fmt.Printf("current context `%s` does not have a namespace selected and `--namespace` flag is empty, exiting", contextName)
		os.Exit(1)
	}

	fmt.Printf("current context is: %s\n", currentContext.AuthInfo)

	// create export directory if it doesnt exist
	err = os.MkdirAll(filepath.Join(o.ExportDir, o.Namespace), 0700)
	switch {
	case os.IsExist(err):
	case err != nil:
		fmt.Printf("error creating the export directory: %#v", err)
		os.Exit(1)
	}

	// create export directory if it doesnt exist
	err = os.Mkdir(filepath.Join(o.ExportDir, o.Namespace, "resources"), 0700)
	switch {
	case os.IsExist(err):
	case err != nil:
		fmt.Printf("error creating the resources directory: %#v", err)
		os.Exit(1)
	}

	// create export directory if it doesnt exist
	err = os.Mkdir(filepath.Join(o.ExportDir, o.Namespace, "failures"), 0700)
	switch {
	case os.IsExist(err):
	case err != nil:
		fmt.Printf("error creating the failures directory: %#v", err)
		os.Exit(1)
	}

	discoveryclient, err := o.configFlags.ToDiscoveryClient()
	if err != nil {
		fmt.Printf("cannot create discovery client: %#v", err)
		os.Exit(1)
	}

	// Always request fresh data from the server
	discoveryclient.Invalidate()

	restConfig, err := o.configFlags.ToRESTConfig()
	if err != nil {
		fmt.Printf("cannot create rest config: %#v", err)
	}

	dynamicClient := dynamic.NewForConfigOrDie(restConfig)

	errs := []error{}
	lists, err := discoveryclient.ServerPreferredNamespacedResources()
	if err != nil {
		fmt.Printf("unauthorized to get discovery service resources: %#v", err)
		return err
	}

	resources, resourceErrs := resourceToExtract(o.Namespace, dynamicClient, lists)
	for _, e := range errs {
		fmt.Printf("error exporting resource: %#v\n", e)
	}

	errs = writeResources(resources, filepath.Join(o.ExportDir, o.Namespace, "resources"))
	for _, e := range errs {
		fmt.Printf("error writing maniffest to file: %#v\n", e)
	}

	errs = writeErrors(resourceErrs, filepath.Join(o.ExportDir, o.Namespace, "failures"))

	return errorsutil.NewAggregate(errs)
}
