package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToHCLSingle(t *testing.T) {
	yaml := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  TEST: test`

	r := strings.NewReader(yaml)
	output, err := ToHCL(r, "", false)

	if err != nil {
		t.Fatal("Converting to HCL failed:", err)
	}

	expected := `
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
}`

	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(output))
}

func TestToHCLMultiple(t *testing.T) {
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
	output, err := ToHCL(r, "", false)

	if err != nil {
		t.Fatal("Converting to HCL failed:", err)
	}

	expected := `
resource "kubernetes_manifest_hcl" "configmap_one" {
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

resource "kubernetes_manifest_hcl" "configmap_two" {
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

func TestToHCLProviderAlias(t *testing.T) {
	yaml := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  TEST: test`

	r := strings.NewReader(yaml)
	output, err := ToHCL(r, "kubernetes-alpha", false)

	if err != nil {
		t.Fatal("Converting to HCL failed:", err)
	}

	expected := `
resource "kubernetes_manifest_hcl" "configmap_test" {
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

func TestToHCLProviderServerSideStrip(t *testing.T) {
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
  uid: bea6500b-0637-4d2d-b726-e0bda0b595dd`

	r := strings.NewReader(yaml)
	output, err := ToHCL(r, "", true)

	if err != nil {
		t.Fatal("Converting to HCL failed:", err)
	}

	expected := `
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
}`

	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(output))
}
