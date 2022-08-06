tfk8s [![Go Report Card](https://goreportcard.com/badge/github.com/jrhouston/tfk8s)](https://goreportcard.com/report/github.com/jrhouston/tfk8s) ![tests](https://github.com/jrhouston/tfk8s/actions/workflows/test.yaml/badge.svg)

---

![](https://media.giphy.com/media/g8GfH3i5F0hby/giphy.gif)

`tfk8s` is a tool that makes it easier to work with the [Terraform Kubernetes Provider](https://github.com/hashicorp/terraform-provider-kubernetes).

If you want to copy examples from the Kubernetes documentation or migrate existing YAML manifests and use them with Terraform without having to convert YAML to HCL by hand, this tool is for you.

## Demo

[<img src="https://asciinema.org/a/jSmyAg4Ar6EcwKCTCXN8iAJM2.svg" width="250">](https://asciinema.org/a/jSmyAg4Ar6EcwKCTCXN8iAJM2)

## Features

- Convert a YAML file containing multiple manifests
- Strip out server side fields when piping `kubectl get $R -o yaml | tfk8s --strip`

## Install

```
go install github.com/jrhouston/tfk8s@latest
```

Alternatively, clone this repo and run:

```
make install
```

If Go's bin directory is not in your `PATH` you will need to add it:

```
export PATH=$PATH:$(go env GOPATH)/bin
```

Or you can install via [brew](https://formulae.brew.sh/formula/tfk8s) for macOS/Linux:

```
brew install tfk8s
```

On macOS, you can also install via [MacPorts](https://www.macports.org):

```
sudo port install tfk8s
```

## Usage

```
Usage of tfk8s:
  -f, --file string         Input file containing Kubernetes YAML manifests (default "-")
  -M, --map-only            Output only an HCL map structure
  -o, --output string       Output file to write Terraform config (default "-")
  -p, --provider provider   Provider alias to populate the provider attribute
  -s, --strip               Strip out server side fields - use if you are piping from kubectl get
  -Q, --strip-key-quotes    Strip out quotes from HCL map keys unless they are required.
  -V, --version             Show tool version
```

## Examples

### Create Terraform configuration from YAML files

```
tfk8s -f input.yaml -o output.tf
```

or, using pipes:
```
cat input.yaml | tfk8s > output.tf
```

**input.yaml**:
```yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  TEST: test
```

✨✨ magically becomes ✨✨

**output.tf**:
```hcl
resource "kubernetes_manifest" "configmap_test" {
  manifest = {
    "apiVersion" = "v1"
    "data" = {
      "TEST" = "test"
    }
    "kind" = "ConfigMap"
    "metadata" = {
      "name" = "test"
    }
  }
}
```

### Use with kubectl to output maps instead of YAML

```
kubectl get ns default -o yaml | tfk8s -M
```
```hcl
{
  "apiVersion" = "v1"
  "kind" = "Namespace"
  "metadata" = {
    "creationTimestamp" = "2020-05-02T15:01:32Z"
    "name" = "default"
    "resourceVersion" = "147"
    "selfLink" = "/api/v1/namespaces/default"
    "uid" = "6ac3424c-07a4-4a69-86ae-cc7a4ae72be3"
  }
  "spec" = {
    "finalizers" = [
      "kubernetes",
    ]
  }
  "status" = {
    "phase" = "Active"
  }
}
```

### Convert a Helm chart to Terraform

You can use `helm template` to generate a manifest from the chart, then pipe it into tfk8s:


```
helm template ./chart-path -f values.yaml | tfk8s
```
