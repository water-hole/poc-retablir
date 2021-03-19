package transform

import (
	"fmt"
	"io/ioutil"
	"os"

	tf "github.com/konveyor/transformations/pkg/transform"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

type TransformOptions struct {
	ExportDir        string
	TransformDir     string
	OldImageRegistry string
	NewImageRegistry string
	IsOpenshift      bool
}

func (o *TransformOptions) Complete(c *cobra.Command, args []string) error {
	// TODO: shawn-hurley
	return nil
}

func (o *TransformOptions) Validate() error {
	// TODO: shawn-hurley
	return nil
}

func (o *TransformOptions) Run() error {
	return o.run()
}

func NewTransformCommand() *cobra.Command {
	o := &TransformOptions{}
	cmd := &cobra.Command{
		Use:   "transform",
		Short: "Create the transformations for the given output",
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

func addFlagsForOptions(o *TransformOptions, cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.ExportDir, "export-dir", "export", "The path to the exported files")
	cmd.Flags().StringVar(&o.TransformDir, "transform-dir", "transform", "The path for output of the transformations")
	cmd.Flags().StringVar(&o.OldImageRegistry, "old-image-registry", "", "The image registry that should be replaced")
	cmd.Flags().StringVar(&o.NewImageRegistry, "new-image-registry", "", "The image registry that will be set in the transform")
	cmd.Flags().BoolVarP(&o.IsOpenshift, "from-openshift", "0", false, "Is the exported files coming from openshift")

	//TODO: Handle adding annotations
}

func (o *TransformOptions) run() error {
	files, err := ioutil.ReadDir(o.ExportDir)
	if err != nil {
		return err
	}
	jsonArray := readFiles(o.ExportDir, files)
	//TODO: writing the files should not be handled in the library.
	return tf.OutputTransforms(jsonArray, tf.TransformOptions{
		IsOpenshift:         o.IsOpenshift,
		StartingPath:        o.ExportDir,
		OutputDirPath:       o.TransformDir,
		OldInternalRegistry: o.OldImageRegistry,
		NewInternalRegistry: o.NewImageRegistry,
	})
}

func readFiles(path string, files []os.FileInfo) []tf.TransformFile {
	jsonFiles := []tf.TransformFile{}
	for _, file := range files {
		filePath := fmt.Sprintf("%v/%v", path, file.Name())
		if file.IsDir() {
			newFiles, err := ioutil.ReadDir(filePath)
			if err != nil {
				fmt.Printf("%v", err)
			}
			files := readFiles(filePath, newFiles)
			jsonFiles = append(jsonFiles, files...)
		} else {
			data, err := ioutil.ReadFile(filePath)
			if err != nil {
				fmt.Printf("%v", err)
			}
			json, err := yaml.YAMLToJSON(data)
			if err != nil {
				fmt.Printf("%v", err)
			}

			u := unstructured.Unstructured{}
			u.UnmarshalJSON(json)

			jsonFiles = append(jsonFiles, tf.TransformFile{
				FileInfo:     file,
				Path:         filePath,
				Unstructured: u,
			})
		}
	}
	return jsonFiles
}
