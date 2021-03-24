package export

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

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

func resourceToExtract(namespace string, dynamicClient dynamic.Interface, lists []*metav1.APIResourceList) ([]*groupResource, []*groupResourceError) {
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

			objs, err := getObjects(g, namespace, dynamicClient)
			if err != nil {
				switch {
				case apierrors.IsForbidden(err):
					fmt.Printf("cannot list obj in namespace\n")
				case apierrors.IsMethodNotSupported(err):
					fmt.Printf("list method not supported on the gvr\n")
				case apierrors.IsNotFound(err):
					fmt.Printf("could not find the resource, most likely this is a virtual resource\n")
				default:
					fmt.Printf("error listing objects: %#v\n", err)
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
}
