package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestYAMLToTerraformResourcesSingle(t *testing.T) {
	yaml := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  TEST: test`

	r := strings.NewReader(yaml)
	output, err := YAMLToTerraformResources(r, "", false, false)

	if err != nil {
		t.Fatal("Converting to HCL failed:", err)
	}

	expected := `
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
}`

	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(output))
}

func TestYAMLToTerraformResourcesEscapeShell(t *testing.T) {
	yaml := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  SCRIPT: |
    echo Hello, ${USER} your homedir is ${HOME}`

	r := strings.NewReader(yaml)
	output, err := YAMLToTerraformResources(r, "", false, false)

	if err != nil {
		t.Fatal("Converting to HCL failed:", err)
	}

	expected := `
resource "kubernetes_manifest" "configmap_test" {
  manifest = {
    "apiVersion" = "v1"
    "data" = {
      "SCRIPT" = "echo Hello, $${USER} your homedir is $${HOME}"
    }
    "kind" = "ConfigMap"
    "metadata" = {
      "name" = "test"
    }
  }
}`

	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(output))
}

func TestYAMLToTerraformResourcesMultiple(t *testing.T) {
	yaml := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: one
data:
  TEST: one
---
# this empty
# document 
# should be 
# skipped
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: two
data:
  TEST: two`

	r := strings.NewReader(yaml)
	output, err := YAMLToTerraformResources(r, "", false, false)

	if err != nil {
		t.Fatal("Converting to HCL failed:", err)
	}

	expected := `
resource "kubernetes_manifest" "configmap_one" {
  manifest = {
    "apiVersion" = "v1"
    "data" = {
      "TEST" = "one"
    }
    "kind" = "ConfigMap"
    "metadata" = {
      "name" = "one"
    }
  }
}

resource "kubernetes_manifest" "configmap_two" {
  manifest = {
    "apiVersion" = "v1"
    "data" = {
      "TEST" = "two"
    }
    "kind" = "ConfigMap"
    "metadata" = {
      "name" = "two"
    }
  }
}`

	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(output))
}

func TestYAMLToTerraformResourcesList(t *testing.T) {
	yaml := `---
apiVersion: v1
kind: ConfigMapList
items:
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: one
  data:
    TEST: one
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: two
  data:
    TEST: two
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: two
    namespace: othernamespace
  data:
    TEST: two
`

	r := strings.NewReader(yaml)
	output, err := YAMLToTerraformResources(r, "", false, false)

	if err != nil {
		t.Fatal("Converting to HCL failed:", err)
	}

	expected := `
resource "kubernetes_manifest" "configmap_one" {
  manifest = {
    "apiVersion" = "v1"
    "data" = {
      "TEST" = "one"
    }
    "kind" = "ConfigMap"
    "metadata" = {
      "name" = "one"
    }
  }
}

resource "kubernetes_manifest" "configmap_two" {
  manifest = {
    "apiVersion" = "v1"
    "data" = {
      "TEST" = "two"
    }
    "kind" = "ConfigMap"
    "metadata" = {
      "name" = "two"
    }
  }
}

resource "kubernetes_manifest" "configmap_othernamespace_two" {
  manifest = {
    "apiVersion" = "v1"
    "data" = {
      "TEST" = "two"
    }
    "kind" = "ConfigMap"
    "metadata" = {
      "name" = "two"
      "namespace" = "othernamespace"
    }
  }
}`

	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(output))
}

func TestYAMLToTerraformResourcesProviderAlias(t *testing.T) {
	yaml := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  TEST: test`

	r := strings.NewReader(yaml)
	output, err := YAMLToTerraformResources(r, "kubernetes-alpha", false, false)

	if err != nil {
		t.Fatal("Converting to HCL failed:", err)
	}

	expected := `
resource "kubernetes_manifest" "configmap_test" {
  provider = kubernetes-alpha

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
}`

	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(output))
}

func TestYAMLToTerraformResourcesProviderServerSideStrip(t *testing.T) {
	yaml := `---
apiVersion: v1
data:
  TEST: test
kind: ConfigMap
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"v1","data":{"TEST":"prod"},"test":"ConfigMap","metadata":{"annotations":{},"name":"test","namespace":"default"}}
  creationTimestamp: "2020-04-30T20:34:59Z"
  name: test
  namespace: default
  resourceVersion: "677134"
  selfLink: /api/v1/namespaces/default/configmaps/test
  uid: bea6500b-0637-4d2d-b726-e0bda0b595dd
  finalizers:
  - test`

	r := strings.NewReader(yaml)
	output, err := YAMLToTerraformResources(r, "", true, false)

	if err != nil {
		t.Fatal("Converting to HCL failed:", err)
	}

	expected := `
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
}`

	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(output))
}

func TestYAMLToTerraformResourcesMapOnly(t *testing.T) {
	yaml := `---
apiVersion: v1
data:
  TEST: test
kind: ConfigMap
metadata:
  name: test
  namespace: default
  resourceVersion: "677134"
  selfLink: /api/v1/namespaces/default/configmaps/test
  uid: bea6500b-0637-4d2d-b726-e0bda0b595dd`

	r := strings.NewReader(yaml)
	output, err := YAMLToTerraformResources(r, "", true, true)

	if err != nil {
		t.Fatal("Converting to HCL failed:", err)
	}

	expected := `{
  "apiVersion" = "v1"
  "data" = {
    "TEST" = "test"
  }
  "kind" = "ConfigMap"
  "metadata" = {
    "name" = "test"
  }
}`

	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(output))
}

func TestYAMLToTerraformResourcesEmptyDocSkip(t *testing.T) {
	yaml := `---
apiVersion: v1
data:
  TEST: test
kind: ConfigMap
metadata:
  name: test
  namespace: default
  resourceVersion: "677134"
  selfLink: /api/v1/namespaces/default/configmaps/test
  uid: bea6500b-0637-4d2d-b726-e0bda0b595dd
---

---
apiVersion: v1
data:
  TEST: test
kind: ConfigMap
metadata:
  name: test2
  namespace: default
  resourceVersion: "677134"
  selfLink: /api/v1/namespaces/default/configmaps/test
  uid: bea6500b-0637-4d2d-b726-e0bda0b595dd`

	r := strings.NewReader(yaml)
	output, err := YAMLToTerraformResources(r, "", true, false)

	if err != nil {
		t.Fatal("Converting to HCL failed:", err)
	}

	expected := `resource "kubernetes_manifest" "configmap_test" {
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

resource "kubernetes_manifest" "configmap_test2" {
  manifest = {
    "apiVersion" = "v1"
    "data" = {
      "TEST" = "test"
    }
    "kind" = "ConfigMap"
    "metadata" = {
      "name" = "test2"
    }
  }
}
`

	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(output))
}
