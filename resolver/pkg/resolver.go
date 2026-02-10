/*
Copyright 2020 Adobe. All rights reserved.
This file is licensed to you under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License. You may obtain a copy
of the License at http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under
the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
OF ANY KIND, either express or implied. See the License for the specific language
governing permissions and limitations under the License.
*/
package resolver

import (
	"fmt"
	"io"
	"strings"

	yamlenc "github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// ResolveImages reads yaml or json stream from in, deserialize it and replace images with ones specified in imagemap
// and serialize it back into out stream.
func ResolveImages(in io.Reader, out io.Writer, imgmap map[string]string) error {
	pt := imageTagTransformer{images: imgmap}
	decoder := yaml.NewYAMLOrJSONDecoder(in, 1024)
	var err error
	firstObj := true
	for err == nil || isEmptyYamlError(err) {
		var obj unstructured.Unstructured
		err = decoder.Decode(&obj)
		if err != nil {
			continue
		}
		if obj.GetName() == "" {
			return fmt.Errorf("Missing metadata.name in object %v", obj)
		}
		if obj.GetKind() == "" {
			return fmt.Errorf("Missing kind in object %v", obj)
		}
		pt.findAndReplaceTag(obj.Object)
		buf, err := yamlenc.Marshal(obj.Object)
		if err != nil {
			return fmt.Errorf("Unable to marshal object %v", obj)
		}
		if firstObj {
			firstObj = false
		} else {
			_, err = out.Write([]byte("---\n"))
			if err != nil {
				return err
			}
		}
		_, err = out.Write(buf)
		if err != nil {
			return err
		}

	}
	if err != io.EOF {
		return err
	}
	return nil
}

func isEmptyYamlError(err error) bool {
	return strings.Contains(err.Error(), "is missing in 'null'")
}

type imageTagTransformer struct {
	images map[string]string
}

/*
findAndReplaceTag replaces the image tags inside one object
It searches the object for container session
then loops though all images inside containers session, finds matched ones and update the tag name
*/
func (pt *imageTagTransformer) findAndReplaceTag(obj map[string]interface{}) error {
	found := false

	// Update [container|spec].image
	paths := []string{"container", "spec"}
	for _, path := range paths {
		_, found = obj[path]
		if found {
			err := pt.updateContainer(obj, path)
			if err != nil {
				return err
			}
		}
	}

	// Update containers.[image, image]
	paths = []string{"containers", "initContainers"}
	for _, path := range paths {
		_, found = obj[path]
		if found {
			err := pt.updateContainers(obj, path)
			if err != nil {
				return err
			}
		}
	}

	if !found {
		return pt.findContainers(obj)
	}
	return nil
}

func (pt *imageTagTransformer) resolveImageName(imagename string) (string, bool) {
	// Try direct lookup first
	if newname, ok := pt.images[imagename]; ok {
		return newname, true
	}

	// If image starts with :, try looking up without the : prefix
	if strings.HasPrefix(imagename, ":") {
		trimmedName := strings.TrimPrefix(imagename, ":")
		if newname, ok := pt.images[trimmedName]; ok {
			return newname, true
		}
	}

	// If image starts with @, try looking up without the @ prefix
	if strings.HasPrefix(imagename, "@") {
		trimmedName := strings.TrimPrefix(imagename, "@")
		if newname, ok := pt.images[trimmedName]; ok {
			return newname, true
		}
	}

	return "", false
}

// In Sciences, some apps set an image URL as app env variables (e.g. trampoline)
// resolver needs to be able to resolve bazel targets referenced in the env KV pair
func (pt *imageTagTransformer) updateContainerEnv(container map[string]interface{}) {
	env, found := container["env"].([]interface{})
	if !found {
		return
	}
	for _, kvInterface := range env {
		kv, kvOk := kvInterface.(map[string]interface{})
		if !kvOk {
			return
		}
		envName, nameOk := kv["name"].(string)
		if !nameOk {
			return
		}
		// The pattern so far sets the env var name suffix as 'IMAGE_URL'
		if strings.Contains(envName, "IMAGE_URL") {
			envValue, valueOk := kv["value"].(string)
			if !valueOk {
				return
			}
			// Check if it's a valid Bazel target
			if strings.HasPrefix(envValue, "//") {
				kv["value"] = pt.images[envValue]
			}
			if strings.HasPrefix(envValue, ":") {
				kv["value"] = pt.images[envValue[1:]]
			}
		}
	}
}

func (pt *imageTagTransformer) updateContainers(obj map[string]interface{}, path string) error {
	containers := obj[path].([]interface{})
	for i := range containers {
		container := containers[i].(map[string]interface{})
		pt.updateContainerEnv(container)
		image, found := container["image"]
		if !found {
			continue
		}
		imagename, imagenameOk := image.(string)
		if imagenameOk {
			if newname, ok := pt.resolveImageName(imagename); ok {
				container["image"] = newname
				continue
			}
			if strings.HasPrefix(imagename, "//") {
				return fmt.Errorf("Unresolved image found: %s", imagename)
			}
		}
	}
	return nil
}

func (pt *imageTagTransformer) updateContainer(obj map[string]interface{}, path string) error {
	container := obj[path].(map[string]interface{})
	pt.updateContainerEnv(container)
	image, found := container["image"]
	if found {
		imagename, imagenameOk := image.(string)
		if imagenameOk {
			if strings.HasPrefix(imagename, "//") {
				return fmt.Errorf("unresolved image found: %s", imagename)
			}
			if newname, ok := pt.resolveImageName(imagename); ok {
				container["image"] = newname
			}
		}
	}
	return nil
}

func (pt *imageTagTransformer) findContainers(obj map[string]interface{}) error {
	for key := range obj {
		switch typedV := obj[key].(type) {
		case map[string]interface{}:
			err := pt.findAndReplaceTag(typedV)
			if err != nil {
				return err
			}
		case []interface{}:
			for i := range typedV {
				item := typedV[i]
				typedItem, ok := item.(map[string]interface{})
				if ok {
					err := pt.findAndReplaceTag(typedItem)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
