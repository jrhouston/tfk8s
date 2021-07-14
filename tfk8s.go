package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	flag "github.com/spf13/pflag"

	cty "github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	yaml "sigs.k8s.io/yaml"

	"github.com/jrhouston/tfk8s/contrib/hashicorp/terraform"
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

// yamlToHCL converts a single YAML document Terraform HCL
func yamlToHCL(doc cty.Value, providerAlias string, stripServerSide bool, mapOnly bool) (string, error) {
	var name, resourceName string
	m := doc.AsValueMap()
	kind := m["kind"].AsString()
	if kind != "List" {
		metadata := m["metadata"].AsValueMap()
		name = metadata["name"].AsString()
		re := regexp.MustCompile(`\W`)
		name = strings.ToLower(re.ReplaceAllString(name, "_"))
		resourceName = strings.ToLower(kind) + "_" + name
	} else if !mapOnly {
		return "", fmt.Errorf("Converting v1.List to a full Terraform configuation is currently not supported")
	}

	if stripServerSide {
		doc = stripServerSideFields(doc)
	}
	s := terraform.FormatValue(doc, 0)

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

var yamlSeparator = "\n---"

// YAMLToTerraformResources takes a file containing one or more Kubernetes configs
// and converts it to resources that can be used by the Terraform Kubernetes Provider
func YAMLToTerraformResources(r io.Reader, providerAlias string, stripServerSide bool, mapOnly bool) (string, error) {
	hcl := ""

	buf := bytes.Buffer{}
	_, err := buf.ReadFrom(r)
	if err != nil {
		return "", err
	}

	count := 0
	manifest := string(buf.Bytes())
	docs := strings.Split(manifest, yamlSeparator)
	for _, doc := range docs {
		if strings.TrimSpace(doc) == "" {
			// some manifests have empty documents
			continue
		}

		var b []byte
		b, err = yaml.YAMLToJSON([]byte(doc))
		if err != nil {
			return "", err
		}

		t, err := ctyjson.ImpliedType(b)
		if err != nil {
			return "", err
		}

		doc, err := ctyjson.Unmarshal(b, t)
		if err != nil {
			return "", err
		}

		if doc.IsNull() {
			// skip empty YAML docs
			continue
		}

		formatted, err := yamlToHCL(doc, providerAlias, stripServerSide, mapOnly)

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

	hcl, err := YAMLToTerraformResources(file, *providerAlias, *stripServerSide, *mapOnly)
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
