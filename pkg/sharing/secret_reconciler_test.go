// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package sharing

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sg2v1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen2/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	fakectrl "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
)

func Test_SecretReconciler_happyPath_updatesStatus(t *testing.T) {
	//////////////// Setup
	sourceSecret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test-source",
		},
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: []byte(`{"auths":{"server":{"username":"correctUser","password":"correctPassword","auth":"correctAuth"}}}`),
		},
		Type: "kubernetes.io/dockerconfigjson",
	}
	placeholderSecret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "placeholder-secret",
			Namespace:   "test-target",
			Annotations: map[string]string{"secretgen.carvel.dev/image-pull-secret": ""},
		},
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: []byte(`{"auths":{}}`),
		},
		Type: "kubernetes.io/dockerconfigjson",
	}
	secretExport := sg2v1alpha1.SecretExport{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SecretExport",
			APIVersion: "secretgen.k14s.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test-source",
		},
		Spec: sg2v1alpha1.SecretExportSpec{
			ToNamespaces: []string{"test-target"},
		},
	}

	scheme := scheme.Scheme
	scheme.AddKnownTypes(sg2v1alpha1.SchemeGroupVersion, &secretExport)
	scheme.AddKnownTypes(sg2v1alpha1.SchemeGroupVersion, &sg2v1alpha1.SecretExportList{})

	testLogr := TestLogger{T: t}
	secretExports := NewSecretExportsWarmedUp(NewSecretExports(testLogr))
	f8CtrlCli := fakectrl.NewFakeClient(&sourceSecret, &placeholderSecret, &secretExport)
	secretExportReconciler := NewSecretExportReconciler(f8CtrlCli, secretExports, testLogr)
	secretReconciler := NewSecretReconciler(f8CtrlCli, secretExports, testLogr)
	secretExports.WarmUpFunc = secretExportReconciler.WarmUp

	exportRequest := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      secretExport.Name,
			Namespace: secretExport.Namespace,
		},
	}

	secretRequestPlaceholder := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      placeholderSecret.Name,
			Namespace: placeholderSecret.Namespace,
		},
	}

	//////////////// Test Action: run the reconcilers
	secretExportReconciler.WarmUp()
	status, err := secretExportReconciler.Reconcile(context.Background(), exportRequest)
	require.NoError(t, err)
	require.False(t, status.Requeue)
	status, err = secretReconciler.Reconcile(context.Background(), secretRequestPlaceholder)
	require.NoError(t, err)
	require.False(t, status.Requeue)

	//////////////// Test Assertions: assert state post-reconcile
	// We must Get the placeholder secret again as our local copy's auths is still empty.
	placeholderSecretReloaded := &corev1.Secret{}
	err = f8CtrlCli.Get(context.Background(), secretRequestPlaceholder.NamespacedName, placeholderSecretReloaded)

	assert.Equal(t, sourceSecret.Data[".dockerconfigjson"], placeholderSecretReloaded.Data[".dockerconfigjson"])
	assert.NotNil(t, placeholderSecretReloaded.ObjectMeta.Annotations["secretgen.carvel.dev/status"])
	var observedStatus map[string]interface{}
	err = json.Unmarshal([]byte(placeholderSecretReloaded.ObjectMeta.Annotations["secretgen.carvel.dev/status"]), &observedStatus)
	assert.NoError(t, err)

	// TODO: thoughts from others on the least ugly way of asserting status? these are both pretty ugly...
	// option 1: parse the JSON but fight the type system
	assert.Equal(t, fmt.Sprintf("%v/%v", sourceSecret.Namespace, sourceSecret.Name), observedStatus["secretNames"].([]interface{})[0].(string))
	// option 2: scrape the json-string
	assert.Contains(t, placeholderSecretReloaded.ObjectMeta.Annotations["secretgen.carvel.dev/status"], `"ReconcileSucceeded","status":"True"`)
	// option 3 (not shown): make the struct explicitly in golang so that the parsing and type system 'work'

	secretExportReloaded := &sg2v1alpha1.SecretExport{}
	err = f8CtrlCli.Get(context.Background(), exportRequest.NamespacedName, secretExportReloaded)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(secretExport.Status.Conditions))
	assert.Equal(t, 1, len(secretExportReloaded.Status.Conditions))
	assert.Equal(t, secretExportReloaded.Status.FriendlyDescription, "Reconcile succeeded")
}

// TestLogger is a logr.Logger that prints through a testing.T object.
// we wanted more logs than error logs so this is a copypasta fork from
// https://github.com/go-logr/logr/blob/v0.1.0/testing/test.go
// TODO: move this to some helper file rather than inlining it to the bottom of this file.
type TestLogger struct {
	T *testing.T
}

var _ logr.Logger = TestLogger{}

func (TestLogger) Info(msg string, args ...interface{}) {
	fmt.Println(msg, args)
}

func (TestLogger) Enabled() bool {
	return false
}

func (log TestLogger) Error(err error, msg string, args ...interface{}) {
	log.T.Logf("%s: %v -- %v", msg, err, args)
}

func (log TestLogger) V(v int) logr.InfoLogger {
	return log
}

func (log TestLogger) WithName(name string) logr.Logger {
	fmt.Println("logger with name: ", name)
	return log
}

func (log TestLogger) WithValues(values ...interface{}) logr.Logger {
	fmt.Println("logger with values: ", values)
	return log
}
