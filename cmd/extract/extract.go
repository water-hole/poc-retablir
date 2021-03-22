package extract

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	errorsutil "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd/api"
)

type ExtractorOptions struct {
	configFlags *genericclioptions.ConfigFlags

	ExportDir string
	genericclioptions.IOStreams
}

func (o *ExtractorOptions) Complete(c *cobra.Command, args []string) error {
	// TODO: @alpatel
	return nil
}

func (o *ExtractorOptions) Validate() error {
	// TODO: @alpatel
	return nil
}

func (o *ExtractorOptions) Run() error {
	return o.run()
}

// groupResource contains the APIGroup and APIResource
type groupResource struct {
	APIGroup        string
	APIVersion      string
	APIGroupVersion string
	APIResource     metav1.APIResource
	objects         *unstructured.UnstructuredList
}

type sortableResource struct {
	resources []*groupResource
	sortBy    string
}

func (s sortableResource) Len() int { return len(s.resources) }
func (s sortableResource) Swap(i, j int) {
	s.resources[i], s.resources[j] = s.resources[j], s.resources[i]
}
func (s sortableResource) Less(i, j int) bool {
	ret := strings.Compare(s.compareValues(i, j))
	if ret > 0 {
		return false
	} else if ret == 0 {
		return strings.Compare(s.resources[i].APIResource.Name, s.resources[j].APIResource.Name) < 0
	}
	return true
}

func (s sortableResource) compareValues(i, j int) (string, string) {
	switch s.sortBy {
	case "name":
		return s.resources[i].APIResource.Name, s.resources[j].APIResource.Name
	case "kind":
		return s.resources[i].APIResource.Kind, s.resources[j].APIResource.Kind
	}
	return s.resources[i].APIGroup, s.resources[j].APIGroup
}

type groupResourceError struct {
	APIResource metav1.APIResource `json:",inline"`
	Err         error              `json:"error"`
}

func NewExtractCommand(streams genericclioptions.IOStreams) *cobra.Command {
	o := &ExtractorOptions{
		configFlags: genericclioptions.NewConfigFlags(true),

		IOStreams: streams,
	}
	cmd := &cobra.Command{
		Use:   "extract",
		Short: "Extract the namespace resources in an output directory",
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

func addFlagsForOptions(o *ExtractorOptions, cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.ExportDir, "export-dir", "export", "The path where files are to be exported")
}

func (o *ExtractorOptions) run() error {
	config := o.configFlags.ToRawKubeConfigLoader()
	rawConfig, err := config.RawConfig()
	if err != nil {
		fmt.Printf("error in generating raw config")
		os.Exit(1)
	}
	kubecontext := rawConfig.CurrentContext

	if kubecontext == "" {
		fmt.Printf("current kubecontext is empty")
		os.Exit(1)
	}

	var currentContext *api.Context

	for name, ctx := range rawConfig.Contexts {
		if name == kubecontext {
			currentContext = ctx
		}
	}

	if currentContext == nil {
		fmt.Printf("currentContext is nil")
		os.Exit(1)
	}

	if len(currentContext.Namespace) == 0 {
		fmt.Printf("currentContext Namespace is empty ")
		os.Exit(1)
	}

	fmt.Printf("current context is: %s\n", currentContext.AuthInfo)

	// create export directory if it doesnt exist
	err = os.MkdirAll(filepath.Join(o.ExportDir, currentContext.Namespace), 0700)
	switch {
	case os.IsExist(err):
	case err != nil:
		fmt.Printf("error creating the export directory: %#v", err)
		os.Exit(1)
	}

	// create export directory if it doesnt exist
	err = os.Mkdir(filepath.Join(o.ExportDir, currentContext.Namespace, "resources"), 0700)
	switch {
	case os.IsExist(err):
	case err != nil:
		fmt.Printf("error creating the resources directory: %#v", err)
		os.Exit(1)
	}

	// create export directory if it doesnt exist
	err = os.Mkdir(filepath.Join(o.ExportDir, currentContext.Namespace, "failures"), 0700)
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
	lists, err := discoveryclient.ServerPreferredResources()
	if err != nil {
		fmt.Printf("unauthorized to get discovery service resources: %#v", err)
		return err
	}

	resources, resourceErrs := resourceToExtract(currentContext, dynamicClient, lists)
	for _, e := range errs {
		fmt.Printf("error extracting resource: %#v\n", e)
	}

	errs = writeResources(resources, filepath.Join(o.ExportDir, currentContext.Namespace, "resources"))
	for _, e := range errs {
		fmt.Printf("error writing maniffest to file: %#v\n", e)
	}

	errs = writeErrors(resourceErrs, filepath.Join(o.ExportDir, currentContext.Namespace, "failures"))

	return errorsutil.NewAggregate(errs)
}

func writeResources(resources []*groupResource, resourceDir string) []error {
	errs := []error{}
	for _, r := range resources {
		fmt.Printf("%s %s\n", r.APIResource.Name, r.APIGroupVersion)

		kind := r.APIResource.Kind

		if kind == "" {
			continue
		}

		for _, obj := range r.objects.Items {
			path := filepath.Join(resourceDir, getFilePath(obj))
			f, err := os.Create(path)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			encoder := yaml.NewEncoder(f)
			err = encoder.Encode(obj.Object)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			err = encoder.Close()
			if err != nil {
				errs = append(errs, err)
				continue
			}

			err = f.Close()
			if err != nil {
				errs = append(errs, err)
				continue
			}

		}
	}

	return errs
}

func writeErrors(errors []*groupResourceError, failuresDir string) []error {
	errs := []error{}
	for _, r := range errors {
		fmt.Printf("%s\n", r.APIResource.Name)

		kind := r.APIResource.Kind

		if kind == "" {
			continue
		}

		path := filepath.Join(failuresDir, r.APIResource.Name+".yaml")
		f, err := os.Create(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		encoder := yaml.NewEncoder(f)
		err = encoder.Encode(&r)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		err = encoder.Close()
		if err != nil {
			errs = append(errs, err)
			continue
		}

		err = f.Close()
		if err != nil {
			errs = append(errs, err)
			continue
		}
	}

	return errs
}

func getFilePath(obj unstructured.Unstructured) string {
	namespace := obj.GetNamespace()
	if namespace == "" {
		namespace = "clusterscoped"
	}
	return strings.Join([]string{obj.GetKind(), namespace, obj.GetName()}, "_") + ".yaml"
}

func resourceToExtract(currentContext *api.Context, dynamicClient dynamic.Interface, lists []*metav1.APIResourceList) ([]*groupResource, []*groupResourceError) {
	resources := []*groupResource{}
	errors := []*groupResourceError{}

	for _, list := range lists {
		if len(list.APIResources) == 0 {
			continue
		}
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}
		for _, resource := range list.APIResources {
			if len(resource.Verbs) == 0 {
				continue
			}

			if resource.Kind == "Event" {
				fmt.Printf("resource: %s.%s, skipping\n", gv.String(), resource.Kind)
				continue
			}

			if !resource.Namespaced {
				fmt.Printf("resource: %s.%s is clusterscoped, skipping\n", gv.String(), resource.Kind)
				continue
			}

			fmt.Printf("processing resource: %s.%s\n", gv.String(), resource.Kind)

			if !strings.Contains(strings.Join(resource.Verbs, ", "), "create") {
				fmt.Printf("resource: : %s.%s does not support a create verb, skipping\n", gv.String(), resource.Kind)
				continue
			}

			g := &groupResource{
				APIGroup:        gv.Group,
				APIVersion:      gv.Version,
				APIGroupVersion: gv.String(),
				APIResource:     resource,
			}

			objs, err := getObjects(g, currentContext.Namespace, dynamicClient)
			if err != nil {
				switch {
				case apierrors.IsForbidden(err):
					fmt.Printf("cannot list obj in namespace")
				case apierrors.IsMethodNotSupported(err):
					fmt.Printf("list method not supported on the gvr")
				case apierrors.IsNotFound(err):
					fmt.Printf("could not find the resource, most likely this is a virtual resource")
				default:
					fmt.Printf("error listing objects: %#v", err)
				}
				errors = append(errors, &groupResourceError{
					APIResource: resource,
					Err:         err,
				})
				continue
			}

			if len(objs.Items) > 0 {
				g.objects = objs
				fmt.Printf("more than one object found\n")
				resources = append(resources, g)
				continue
			}

			fmt.Printf("0 objects found, skipping\n")
		}
	}

	sort.Stable(sortableResource{resources, "kind"})
	return resources, errors
}

func getObjects(g *groupResource, namespace string, d dynamic.Interface) (*unstructured.UnstructuredList, error) {
	c := d.Resource(schema.GroupVersionResource{
		Group:    g.APIGroup,
		Version:  g.APIVersion,
		Resource: g.APIResource.Name,
	})
	if g.APIResource.Namespaced {
		return c.Namespace(namespace).List(context.Background(), metav1.ListOptions{})
	}
	return &unstructured.UnstructuredList{}, nil
	//return c.List(context.Background(), metav1.ListOptions{})
}
