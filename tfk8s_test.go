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
	output, err := ToHCL(r, "", false, false, false)

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
	output, err := ToHCL(r, "", false, false, false)

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

func TestToHCLProviderAlias(t *testing.T) {
	yaml := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  TEST: test`

	r := strings.NewReader(yaml)
	output, err := ToHCL(r, "kubernetes-alpha", false, false, false)

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
  uid: bea6500b-0637-4d2d-b726-e0bda0b595dd
  finalizers:
  - test`

	r := strings.NewReader(yaml)
	output, err := ToHCL(r, "", true, false, false)

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

func TestToHCLMapOnly(t *testing.T) {
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
	output, err := ToHCL(r, "", true, true, false)

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

func TestToHCLDefaultNamespace(t *testing.T) {
	yaml := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  TEST: test`

	r := strings.NewReader(yaml)
	output, err := ToHCL(r, "", false, false, true)

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
      "namespace" = "default"
    }
  }
}`

	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(output))
}

func TestToHCLOthertNamespace(t *testing.T) {
	yaml := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  namespace: foobar
data:
  TEST: test`

	r := strings.NewReader(yaml)
	output, err := ToHCL(r, "", false, false, true)

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
      "namespace" = "foobar"
    }
  }
}`

	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(output))
}
