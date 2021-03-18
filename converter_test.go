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
	resources, resourcesMap, err := YAMLToTerraformResources(r, "", false, false)

	if err != nil {
		t.Fatal("Converting to HCL failed:", err)
	}

	expected := []string{
		`resource "kubernetes_manifest" "configmap_test" {
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
`}

	expectedMap := map[string][]string{
		"ConfigMap": expected,
	}

	assert.Equal(t, expected, resources)
	assert.Equal(t, expectedMap, resourcesMap)
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
apiVersion: v1
kind: ConfigMap
metadata:
  name: two
data:
  TEST: two`

	r := strings.NewReader(yaml)
	resources, resourcesMap, err := YAMLToTerraformResources(r, "", false, false)

	if err != nil {
		t.Fatal("Converting to HCL failed:", err)
	}

	expected := []string{
		`resource "kubernetes_manifest" "configmap_one" {
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
`,
		`resource "kubernetes_manifest" "configmap_two" {
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
`}

	expectedMap := map[string][]string{
		"ConfigMap": expected,
	}

	assert.Equal(t, resources, expected)
	assert.Equal(t, resourcesMap, expectedMap)
}

func TestYAMLToTerraformResourcesMultipleKinds(t *testing.T) {
	yaml := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: one
data:
  TEST: one
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: two
data:
  TEST: two
---
apiVersion: v1
kind: Secret
metadata:
  name: three
data:
  secret: cGFzc3dvcmQK
---
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  selector:
    app: MyApp
  ports:
    - protocol: TCP
      port: 80
      targetPort: 1337`

	r := strings.NewReader(yaml)
	_, resourcesMap, err := YAMLToTerraformResources(r, "", false, false)

	if err != nil {
		t.Fatal("Converting to HCL failed:", err)
	}

	configMaps := []string{
		`resource "kubernetes_manifest" "configmap_one" {
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
`,
		`resource "kubernetes_manifest" "configmap_two" {
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
`}
	secrets := []string{
		`resource "kubernetes_manifest" "secret_three" {
  manifest = {
    "apiVersion" = "v1"
    "data" = {
      "secret" = "cGFzc3dvcmQK"
    }
    "kind" = "Secret"
    "metadata" = {
      "name" = "three"
    }
  }
}
`}
	services := []string{
		`resource "kubernetes_manifest" "service_my_service" {
  manifest = {
    "apiVersion" = "v1"
    "kind" = "Service"
    "metadata" = {
      "name" = "my-service"
    }
    "spec" = {
      "ports" = [
        {
          "port" = 80
          "protocol" = "TCP"
          "targetPort" = 1337
        },
      ]
      "selector" = {
        "app" = "MyApp"
      }
    }
  }
}
`}

	expectedMap := map[string][]string{
		"ConfigMap": configMaps,
		"Secret":    secrets,
		"Service":   services,
	}

	assert.Equal(t, resourcesMap, expectedMap)
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
	resources, _, err := YAMLToTerraformResources(r, "kubernetes-alpha", false, false)

	if err != nil {
		t.Fatal("Converting to HCL failed:", err)
	}

	expected := []string{
		`resource "kubernetes_manifest" "configmap_test" {
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
}
`}

	assert.Equal(t, expected, resources)
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
	resources, _, err := YAMLToTerraformResources(r, "", true, false)

	if err != nil {
		t.Fatal("Converting to HCL failed:", err)
	}

	expected := []string{
		`resource "kubernetes_manifest" "configmap_test" {
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
`}

	assert.Equal(t, expected, resources)
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
	resources, _, err := YAMLToTerraformResources(r, "", true, true)

	if err != nil {
		t.Fatal("Converting to HCL failed:", err)
	}

	expected := []string{
		`{
  "apiVersion" = "v1"
  "data" = {
    "TEST" = "test"
  }
  "kind" = "ConfigMap"
  "metadata" = {
    "name" = "test"
  }
}
`}

	assert.Equal(t, expected, resources)
}
