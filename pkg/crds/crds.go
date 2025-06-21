package crds

import (
	"embed"
	"fmt"
	"path/filepath"

	"github.com/rancher/wrangler/v3/pkg/yaml"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

const (
	baseDir = "."
	crdKind = "CustomResourceDefinition"
)

var (
	//go:embed yaml
	crdFS embed.FS

	errDuplicate = fmt.Errorf("duplicate CRD")
)

func List() ([]*apiextv1.CustomResourceDefinition, error) {
	crdMap, err := crdsFromDir(baseDir)
	if err != nil {
		return nil, err
	}
	crds := make([]*apiextv1.CustomResourceDefinition, 0, len(crdMap))
	for _, crd := range crdMap {
		crds = append(crds, crd)
	}
	return crds, nil
}

// crdsFromDir recursively traverses the embedded yaml directory and find all CRD yamls.
// cribbed from https://github.com/rancher/rancher/blob/v2.11.2/pkg/crds/crds.go
func crdsFromDir(dirName string) (map[string]*apiextv1.CustomResourceDefinition, error) {
	// read all entries in the embedded directory
	crdFiles, err := crdFS.ReadDir(dirName)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded dir '%s': %w", dirName, err)
	}

	allCRDs := map[string]*apiextv1.CustomResourceDefinition{}
	for _, dirEntry := range crdFiles {
		fullPath := filepath.Join(dirName, dirEntry.Name())
		if dirEntry.IsDir() {
			// if the entry is the dir recurse into that folder to get all crds
			subCRDs, err := crdsFromDir(fullPath)
			if err != nil {
				return nil, err
			}
			for k, v := range subCRDs {
				if _, ok := allCRDs[k]; ok {
					return nil, fmt.Errorf("%w for '%s", errDuplicate, k)
				}
				allCRDs[k] = v
			}
			continue
		}

		// read the file and convert it to a crd object
		file, err := crdFS.Open(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open embedded file '%s': %w", fullPath, err)
		}
		crdObjs, err := yaml.UnmarshalWithJSONDecoder[*apiextv1.CustomResourceDefinition](file)
		if err != nil {
			return nil, fmt.Errorf("failed to convert embedded file '%s' to yaml: %w", fullPath, err)
		}
		for _, crdObj := range crdObjs {
			if crdObj.Kind != crdKind {
				// if the yaml is not a CRD return an error
				return nil, fmt.Errorf("decoded object is not '%s' instead found Kind='%s'", crdKind, crdObj.Kind)
			}
			if _, ok := allCRDs[crdObj.Name]; ok {
				return nil, fmt.Errorf("%w for '%s", errDuplicate, crdObj.Name)
			}
			allCRDs[crdObj.Name] = crdObj
		}
	}
	return allCRDs, nil
}
