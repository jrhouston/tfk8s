package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform/repl"
	"gopkg.in/yaml.v2"

	flag "github.com/spf13/pflag"
)

var toolVersion string

const resourceType = "kubernetes_manifest"

// NOTE The terraform console formatter only supports map[string]interface{}
// but the yaml parser spits out map[interface{}]interface{} so we need to convert

func fixSlice(s []interface{}) []interface{} {
	fixed := []interface{}{}

	for _, v := range s {
		switch v.(type) {
		case map[interface{}]interface{}:
			fixed = append(fixed, fixMap(v.(map[interface{}]interface{})))
		case []interface{}:
			fixed = append(fixed, fixSlice(v.([]interface{})))
		default:
			fixed = append(fixed, v)
		}
	}

	return fixed
}

func fixMap(m map[interface{}]interface{}) map[string]interface{} {
	fixed := map[string]interface{}{}

	for k, v := range m {
		switch v.(type) {
		case map[interface{}]interface{}:
			fixed[k.(string)] = fixMap(v.(map[interface{}]interface{}))
		case []interface{}:
			fixed[k.(string)] = fixSlice(v.([]interface{}))
		default:
			fixed[k.(string)] = v
		}
	}

	return fixed
}

var serverSideMetadataFields = []string{
	"creationTimestamp",
	"resourceVersion",
	"selfLink",
	"uid",
	"managedFields",
	"finalizers",
}

func stripServerSideFields(m map[string]interface{}) {
	delete(m, "status")

	metadata := m["metadata"].(map[string]interface{})
	for _, f := range serverSideMetadataFields {
		delete(metadata, f)
	}
	if v, ok := metadata["namespace"].(string); ok && v == "default" {
		delete(metadata, "namespace")
	}

	annotations, ok := metadata["annotations"].(map[string]interface{})
	if ok {
		delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
		if len(annotations) == 0 {
			delete(metadata, "annotations")
		}
	}

	spec, ok := m["spec"].(map[string]interface{})
	if ok {
		delete(spec, "finalizers")
	}
}

func toHCL(doc map[interface{}]interface{}, providerAlias string, stripServerSide bool, mapOnly bool) (string, error) {
	formattable := fixMap(doc)

	if stripServerSide {
		stripServerSideFields(formattable)
	}

	// TODO need to find a way of ordering the fields in the output
	s, err := repl.FormatResult(formattable)
	if err != nil {
		return "", err
	}

	kind := formattable["kind"].(string)

	var name, resourceName string
	if kind != "List" {
		name = formattable["metadata"].(map[string]interface{})["name"].(string)
		re := regexp.MustCompile("[.-]")
		name = strings.ToLower(re.ReplaceAllString(name, "_"))
		resourceName = strings.ToLower(kind) + "_" + name
	} else if !mapOnly {
		return "", fmt.Errorf("Converting v1.List to a full Terraform configuation is currently not supported")
	}

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

	return hcl, nil
}

// ToHCL converts a file containing one or more Kubernetes configs
// and converts it to resources that can be used by the Terraform Kubernetes Provider
func ToHCL(r io.Reader, providerAlias string, stripServerSide bool, mapOnly bool) (string, error) {
	hcl := ""

	decoder := yaml.NewDecoder(r)

	count := 0
	var err error
	for {
		var doc map[interface{}]interface{}
		err = decoder.Decode(&doc)

		if err != nil {
			if err == io.EOF {
				break
			} else {
				return "", fmt.Errorf("error parsing YAML: %s", err)
			}
		}

		formatted, err := toHCL(doc, providerAlias, stripServerSide, mapOnly)

		if err != nil {
			return "", fmt.Errorf("error converting YAML to HCL: %s", err)
		}

		if count > 0 {
			hcl += "\n"
		}
		hcl += formatted
		count++
	}

	return hcl, nil
}

func main() {
	infile := flag.StringP("file", "f", "-", "Input file containing Kubernetes YAML manifests")
	outfile := flag.StringP("output", "o", "-", "Output file to write Terraform config")
	providerAlias := flag.StringP("provider", "p", "", "Provider alias to populate the `provider` attribute")
	stripServerSide := flag.BoolP("strip", "s", false, "Strip out server side fields - use if you are piping from kubectl get")
	version := flag.BoolP("version", "V", false, "Show tool version")
	mapOnly := flag.BoolP("map-only", "M", false, "Output only an HCL map structure")
	flag.Parse()

	if *version {
		fmt.Println(toolVersion)
		os.Exit(0)
	}

	var file *os.File
	if *infile == "-" {
		file = os.Stdin
	} else {
		var err error
		file, err = os.Open(*infile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\r\n", err.Error())
			os.Exit(1)
		}
	}

	hcl, err := ToHCL(file, *providerAlias, *stripServerSide, *mapOnly)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if *outfile == "-" {
		fmt.Print(hcl)
	} else {
		ioutil.WriteFile(*outfile, []byte(hcl), 0644)
	}
}
