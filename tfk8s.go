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

const resourceType = "kubernetes_manifest_hcl"

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

func toHCL(doc map[interface{}]interface{}) (string, error) {
	hcl := ""

	formattable := fixMap(doc)
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
	infile := flag.StringP("file", "f", "-", "Input file containing Kubernetes YAML manifests")
	outfile := flag.StringP("output", "o", "-", "Output file to write Terraform config")
	flag.Parse()

	var file *os.File
	if *infile == "-" {
		file = os.Stdin
	} else {
		var err error
		file, err = os.Open(*infile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	hcl, err := ToHCL(file)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// TODO use -f for input and -o for output
	// TODO support stripping server-side fields
	// TODO find a way of ordering the keys
	// TODO add flag for adding provider attribute

	if *outfile == "-" {
		fmt.Print(hcl)
	} else {
		ioutil.WriteFile(*outfile, []byte(hcl), 0644)
	}
}
