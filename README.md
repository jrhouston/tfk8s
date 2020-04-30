tfk8s
---

![](https://media.giphy.com/media/g8GfH3i5F0hby/giphy.gif)

`tfk8s` is a tool for converting Kubernetes YAML manifests to HashiCorp's HCL for use the with the [Terraform Kubernetes Provider](https://github.com/hashicorp/terraform-provider-kubernetes-alpha).

If you want to copy examples from the Kubernetes documentation or migrate existing YAML manifests and use them with Terraform without having to convert them to HCL by hand, this tool is for you.

## Install

For the moment, clone this repo and:

```
go install
```

## Usage

```
tfk8s input.yaml output.tf
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
