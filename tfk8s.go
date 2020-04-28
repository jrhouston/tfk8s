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
)

const resourceType = "kubernetes_manifest_hcl"

// NOTE The terraform console formatter only supports map[string]interface{}
// but the yaml parser spits out map[interface{}]interface{} so we need to convert

func fixslice(s []interface{}) []interface{} {
	fixed := []interface{}{}

	for _, v := range s {
		switch v.(type) {
		case map[interface{}]interface{}:
			fixed = append(fixed, fixmap(v.(map[interface{}]interface{})))
		case []interface{}:
			fixed = append(fixed, fixslice(v.([]interface{})))
		default:
			fixed = append(fixed, v)
		}
	}

	return fixed
}

func fixmap(m map[interface{}]interface{}) map[string]interface{} {
	fixed := map[string]interface{}{}

	for k, v := range m {
		switch v.(type) {
		case map[interface{}]interface{}:
			fixed[k.(string)] = fixmap(v.(map[interface{}]interface{}))
		case []interface{}:
			fixed[k.(string)] = fixslice(v.([]interface{}))
		default:
			fixed[k.(string)] = v
		}
	}

	return fixed
}

func toHCL(doc map[interface{}]interface{}) (string, error) {
	hcl := ""

	formattable := fixmap(doc)
	s, err := repl.FormatResult(formattable)
	if err != nil {
		return "", err
	}

	name := formattable["metadata"].(map[string]interface{})["name"].(string)
	re := regexp.MustCompile("[.-]")
	name = strings.ToLower(re.ReplaceAllString(name, "_"))

	kind := formattable["kind"].(string)
	resourceName := strings.ToLower(kind) + "_" + name

	hcl += fmt.Sprintf("resource %q %q {\n", resourceType, resourceName)
	hcl += fmt.Sprintf("  manifest = %v\n", strings.ReplaceAll(s, "\n", "\n  "))
	hcl += fmt.Sprintf("}\n")

	return hcl, nil
}

// ToHCL converts a file containing one or more Kubernetes configs
// and converts it to resources that can be used by the Terraform Kubernetes Provider
func ToHCL(r io.Reader) (string, error) {
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

		formatted, err := toHCL(doc)
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
	if len(os.Args) < 2 {
		fmt.Printf("Usage %s <resource.yaml>\n", os.Args[0])
		os.Exit(1)
	}

	infile := os.Args[1]

	var outfile string
	if len(os.Args) == 3 {
		outfile = os.Args[2]
	}

	f, err := os.Open(infile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	hcl, err := ToHCL(f)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// TODO support stripping server-side fields
	// TODO find a way of ordering the keys

	if outfile != "" {
		ioutil.WriteFile(outfile, []byte(hcl), 0644)
	} else {
		fmt.Print(hcl)
	}
}
