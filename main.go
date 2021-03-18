package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	flag "github.com/spf13/pflag"
)

func main() {
	infile := flag.StringP("file", "f", "", "Input file containing Kubernetes YAML manifests (`-` is STDIN)")
	outfile := flag.StringP("output", "o", "-", "Output file to write Terraform config (`-` is STDOUT)")
	providerAlias := flag.StringP("provider", "p", "", "Provider alias to populate the `provider` attribute")
	stripServerSide := flag.BoolP("strip", "s", false, "Strip out server side fields - use if you are piping from kubectl get")
	version := flag.BoolP("version", "V", false, "Show tool version")
	mapOnly := flag.BoolP("map-only", "M", false, "Output only an HCL map structure")
	split := flag.BoolP("split", "S", false, "Split manifest into files for each resource type (e.g deployment.tf, service.tf)")
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

	resources, resourcesMap, err := YAMLToTerraformResources(file, *providerAlias, *stripServerSide, *mapOnly)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// FIXME add error handling for write failures
	if *split {
		for kind, r := range resourcesMap {
			outfile := fmt.Sprintf("%s.tf", strings.ToLower(kind))
			ioutil.WriteFile(outfile, []byte(strings.Join(r, "\n")), 0644)
		}
	} else {
		hcl := strings.Join(resources, "\n")
		if *outfile == "-" {
			fmt.Print(hcl)
		} else {
			ioutil.WriteFile(*outfile, []byte(hcl), 0644)
		}
	}
}
