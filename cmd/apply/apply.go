package apply

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	tf "github.com/konveyor/transformations/pkg/transform"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

type ApplyOptions struct {
	ExportDir    string
	TransformDir string
	OutputDir    string
}

func (o *ApplyOptions) Complete(c *cobra.Command, args []string) error {
	// TODO: shawn-hurley
	return nil
}

func (o *ApplyOptions) Validate() error {
	// TODO: shawn-hurley
	return nil
}

func (o *ApplyOptions) Run() error {
	return o.run()
}

func NewApplyCommand() *cobra.Command {
	o := &ApplyOptions{}
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Create the files to be applied to the cluster from transformations ",
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

func addFlagsForOptions(o *ApplyOptions, cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.ExportDir, "export-dir", "export", "The path to the exported files")
	cmd.Flags().StringVar(&o.TransformDir, "transform-dir", "transform", "The path for output of the transformations")
	cmd.Flags().StringVar(&o.OutputDir, "output-dir", "output", "The path for output of the transformations")
}

func (o *ApplyOptions) run() error {
	files, err := ioutil.ReadDir(o.ExportDir)
	if err != nil {
		fmt.Printf("%v", err)
	}
	jsonArray := readFiles(o.ExportDir, files)
	for _, file := range jsonArray {
		fmt.Printf("\n")
		fname, _ := tf.GetTransformPath(o.TransformDir, o.ExportDir, file.Path)
		whfname, _ := tf.GetWhiteOutFilePath(o.TransformDir, o.ExportDir, file.Path)

		// if white out alert user, and continue
		_, err := os.Stat(whfname)
		if !os.IsNotExist(err) {
			fmt.Printf("\nSkipping file: %v becuase it should be deleted", file.Path)
			continue
		}

		// Get transform
		patchesJSON, err := ioutil.ReadFile(fname)
		if err != nil {
			fmt.Printf("error: %v", err)
		}

		pa, err := jsonpatch.DecodePatch(patchesJSON)
		if err != nil {
			fmt.Printf("error: %v", err)
		}

		doc, err := file.Unstructured.MarshalJSON()
		//Determine if annoations need to be added
		// ADD CHECK FOR IF NEW ANNOTATIONS
		if len(file.Unstructured.GetAnnotations()) == 0 {
			//Apply patches to doc to add annoations.
			patches := []byte(`[
				{"op": "add", "path": "/metadata/annotations", "value": {}}
			]`)
			patch, err := jsonpatch.DecodePatch(patches)
			if err != nil {
				fmt.Printf("\n unable to decode patch err: %v", err)
			}
			doc, err = patch.Apply(doc)
			if err != nil {
				fmt.Printf("\n unable to apply patch err: %v", err)
			}
		}

		// apply transformation
		output, err := pa.Apply([]byte(doc))
		if err != nil {
			fmt.Printf("can not apply patch err: %v - file: %v", err, file.Path)
		}

		output, err = yaml.JSONToYAML(output)
		if err != nil {
			fmt.Printf("can not convert to yaml %v", err)
		}

		// write file to output
		dir, newName := filepath.Split(file.Path)
		dir = strings.Replace(dir, o.ExportDir, o.OutputDir, 1)
		err = os.MkdirAll(dir, 0777)
		if err != nil {
			fmt.Printf("err: %v", err)
		}
		err = ioutil.WriteFile(filepath.Join(dir, newName), output, 0664)
		if err != nil {
			fmt.Printf("err: %v", err)
		}
	}
	return nil
}

// This needs to be moved to a library function.
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
