tfk8s
---

![](https://media.giphy.com/media/g8GfH3i5F0hby/giphy.gif)

`tfk8s` is a tool that makes it easier to work with the [Terraform Kubernetes Provider](https://github.com/hashicorp/terraform-provider-kubernetes-alpha).

If you want to copy examples from the Kubernetes documentation or migrate existing YAML manifests and use them with Terraform without having to convert YAML to HCL by hand, this tool is for you. 

## Demo 

[<img src="https://asciinema.org/a/5VZoVK03s9BYSLLWUZd6XmAxx.svg" width="250">](https://asciinema.org/a/5VZoVK03s9BYSLLWUZd6XmAxx)

## Features

- Convert a YAML file containing multiple manifests
- Strip out server side fields when piping `kubectl get $R -o yaml | tfk8s --strip`

## Install

Grab the binary from the [releases](https://github.com/jrhouston/tfk8s/releases) page.

Or clone this repo and:

```
make install
```

## Usage

### Creating Terraform configurations

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
resource "kubernetes_manifest_hcl" "configmap_test" {
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
