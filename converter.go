package main

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform/repl"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"sigs.k8s.io/yaml"
)

// toolVersion is the version that gets printed when you run --version
var toolVersion string

// resourceType is the type of Terraform resource
var resourceType = "kubernetes_manifest"

// ignoreMetadata is the list of metadata fields to strip
// when --strip is supplied
var ignoreMetadata = []string{
	"creationTimestamp",
	"resourceVersion",
	"selfLink",
	"uid",
	"managedFields",
	"finalizers",
}

// ignoreAnnotations is the list of annotations to strip
// when --strip is supplied
var ignoreAnnotations = []string{
	"kubectl.kubernetes.io/last-applied-configuration",
}

// stripServerSideFields removes fields that have been added on the
// server side after the resource was created such as the status field
func stripServerSideFields(doc cty.Value) cty.Value {
	m := doc.AsValueMap()

	// strip server-side metadata
	metadata := m["metadata"].AsValueMap()
	for _, f := range ignoreMetadata {
		delete(metadata, f)
	}
	if v, ok := metadata["annotations"]; ok {
		annotations := v.AsValueMap()
		for _, a := range ignoreAnnotations {
			delete(annotations, a)
		}
		if len(annotations) == 0 {
			delete(metadata, "annotations")
		} else {
			metadata["annotations"] = cty.ObjectVal(annotations)
		}
	}
	if ns, ok := metadata["namespace"]; ok && ns.AsString() == "default" {
		delete(metadata, "namespace")
	}
	m["metadata"] = cty.ObjectVal(metadata)

	// strip finalizer from spec
	if v, ok := m["spec"]; ok {
		mm := v.AsValueMap()
		delete(mm, "finalizers")
		m["spec"] = cty.ObjectVal(mm)
	}

	// strip status field
	delete(m, "status")

	return cty.ObjectVal(m)
}

// ctyToHCL takes a cty.Value containing a Kubernetes resource and returns the corresponding HCL and it's Kind
func ctyToHCL(v cty.Value, providerAlias string, stripServerSide bool, mapOnly bool) (string, string, error) {
	var name, resourceName string
	m := v.AsValueMap()
	kind := m["kind"].AsString()
	if kind != "List" {
		metadata := m["metadata"].AsValueMap()
		name = metadata["name"].AsString()
		re := regexp.MustCompile(`\W`)
		name = strings.ToLower(re.ReplaceAllString(name, "_"))
		resourceName = strings.ToLower(kind) + "_" + name
	} else if !mapOnly {
		return "", "", fmt.Errorf("Converting v1.List to a full Terraform configuation is currently not supported")
	}

	if stripServerSide {
		v = stripServerSideFields(v)
	}
	s := repl.FormatValue(v, 0)

	var hcl string
	if mapOnly {
		hcl = fmt.Sprintf("%v\n", s)
	} else {
		hcl = fmt.Sprintf("resource %q %q {\n", resourceType, resourceName)
		if providerAlias != "" {
			hcl += fmt.Sprintf("  provider = %v\n\n", providerAlias)
		}
		hcl += fmt.Sprintf("  manifest = %v\n", strings.ReplaceAll(s, "\n", "\n  "))
		hcl += fmt.Sprintf("}\n")
	}

	return hcl, kind, nil
}

var yamlSeparator = "\n---"

// yamlToCty converts a string containing one YAML document to a cty.Value
func yamlToCty(doc string) (cty.Value, error) {
	// this sucks but basically we just convert the YAML
	// to JSON then run it through ctyjson.Marshal
	b, err := yaml.YAMLToJSON([]byte(doc))
	if err != nil {
		return cty.NilVal, err
	}

	t, err := ctyjson.ImpliedType(b)
	if err != nil {
		return cty.NilVal, err
	}

	v, err := ctyjson.Unmarshal(b, t)
	if err != nil {
		return cty.NilVal, err
	}
	return v, nil
}

// YAMLToTerraformResources takes a file containing one or more Kubernetes configs
// and converts it to resources that can be used by the Terraform Kubernetes Provider.
//
// This function returns both a string slice with the HCL of each resource and a
// map which buckets the HCL for each resource by its Kind.
func YAMLToTerraformResources(r io.Reader, providerAlias string, stripServerSide bool, mapOnly bool) ([]string, map[string][]string, error) {
	resources := []string{}
	resourcesMap := map[string][]string{}

	buf := bytes.Buffer{}
	_, err := buf.ReadFrom(r)
	if err != nil {
		return nil, nil, err
	}

	manifest := string(buf.Bytes())
	docs := strings.Split(manifest, yamlSeparator)
	for _, doc := range docs {
		v, err := yamlToCty(doc)
		if err != nil {
			return nil, nil, fmt.Errorf("error converting YAML to HCL: %s", err)
		}
		resource, kind, err := ctyToHCL(v, providerAlias, stripServerSide, mapOnly)
		if err != nil {
			return nil, nil, err
		}

		resources = append(resources, resource)
		resourcesMap[kind] = append(resourcesMap[kind], resource)
	}

	return resources, resourcesMap, nil
}
