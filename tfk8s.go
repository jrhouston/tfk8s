package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"runtime/debug"
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

// snakify converts "a-String LIKE this" to "a_string_like_this"
func snakify(s string) string {
	re := regexp.MustCompile(`\W`)
	return strings.ToLower(re.ReplaceAllString(s, "_"))
}

// escape incidences of ${} with $${} to prevent Terraform trying to interpolate them
func escapeShellVars(s string) string {
	r := regexp.MustCompile(`(\${.*?})`)
	return r.ReplaceAllString(s, `$$$1`)
}

// yamlToHCL converts a single YAML document Terraform HCL
func yamlToHCL(doc cty.Value, providerAlias string, stripServerSide bool, mapOnly bool) (string, error) {
	m := doc.AsValueMap()
	docs := []cty.Value{doc}
	if strings.HasSuffix(m["kind"].AsString(), "List") {
		docs = m["items"].AsValueSlice()
	}

	hcl := ""
	for i, doc := range docs {
		mm := doc.AsValueMap()
		kind := mm["kind"].AsString()
		metadata := mm["metadata"].AsValueMap()
		name := metadata["name"].AsString()
		var namespace string
		if v, ok := metadata["namespace"]; ok {
			namespace = v.AsString()
		}

		resourceName := kind
		if namespace != "" && namespace != "default" {
			resourceName = resourceName + "_" + namespace
		}
		resourceName = resourceName + "_" + name
		resourceName = snakify(resourceName)

		if stripServerSide {
			doc = stripServerSideFields(doc)
		}
		s := terraform.FormatValue(doc, 0)
		s = escapeShellVars(s)

		if mapOnly {
			hcl += fmt.Sprintf("%v\n", s)
		} else {
			hcl += fmt.Sprintf("resource %q %q {\n", resourceType, resourceName)
			if providerAlias != "" {
				hcl += fmt.Sprintf("  provider = %v\n\n", providerAlias)
			}
			hcl += fmt.Sprintf("  manifest = %v\n", strings.ReplaceAll(s, "\n", "\n  "))
			hcl += fmt.Sprintf("}\n")
		}
		if i != len(docs)-1 {
			hcl += "\n"
		}
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

		if !doc.Type().IsObjectType() {
			return "", fmt.Errorf("the manifest must be a YAML document")
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

func capturePanic() {
	if r := recover(); r != nil {
		fmt.Printf(
			"panic: %s\n\n%s\n\n"+
				"⚠️  Oh no! Looks like your manifest caused tfk8s to crash.\n\n"+
				"Please open a GitHub issue and include your manifest YAML with the stack trace above,\n"+
				"or ping me on slack and I'll try and fix it!\n\n"+
				"GitHub: https://github.com/jrhouston/tfk8s/issues\n"+
				"Slack: #terraform-providers on https://kubernetes.slack.com\n\n"+
				"- Thanks, @jrhouston\n\n",
			r, debug.Stack())
	}
}

func main() {
	defer capturePanic()

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
		fmt.Println("error:", err)
		os.Exit(1)
	}

	if *outfile == "-" {
		fmt.Print(hcl)
	} else {
		ioutil.WriteFile(*outfile, []byte(hcl), 0644)
	}
}
