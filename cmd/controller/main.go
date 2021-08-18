// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package main

// Based on https://github.com/kubernetes-sigs/controller-runtime/blob/8f633b179e1c704a6e40440b528252f147a3362a/examples/builtins/main.go

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	sgv1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen/v1alpha1"
	sgclient "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/client/clientset/versioned"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/generator"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/sharing"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// Version of secretgen-controller that will be visible to users
	Version = "0.4.0-alpha.1"
)

var (
	log           = logf.Log.WithName("sg")
	ctrlNamespace = ""
)

func main() {
	flag.StringVar(&ctrlNamespace, "namespace", "", "Namespace to watch")
	flag.Parse()

	logf.SetLogger(zap.New(zap.UseDevMode(false)))
	entryLog := log.WithName("entrypoint")
	entryLog.Info("secretgen-controller", "version", Version)

	entryLog.Info("setting up manager")
	restConfig := config.GetConfigOrDie()

	// Register sg API types so they can be watched
	sgv1alpha1.AddToScheme(scheme.Scheme)

	mgr, err := manager.New(restConfig, manager.Options{Namespace: ctrlNamespace})
	exitIfErr(entryLog, "unable to set up controller manager", err)

	entryLog.Info("setting up controllers")

	coreClient, err := kubernetes.NewForConfig(restConfig)
	exitIfErr(entryLog, "building core client", err)

	sgClient, err := sgclient.NewForConfig(restConfig)
	exitIfErr(entryLog, "building secretgen client", err)

	certReconciler := generator.NewCertificateReconciler(sgClient, coreClient, log.WithName("cert"))
	_, err = registerCtrl("cert", mgr, certReconciler, &source.Kind{Type: &sgv1alpha1.Certificate{}})
	exitIfErr(entryLog, "registering certificate controller", err)

	passwordReconciler := generator.NewPasswordReconciler(sgClient, coreClient, log.WithName("password"))
	_, err = registerCtrl("password", mgr, passwordReconciler, &source.Kind{Type: &sgv1alpha1.Password{}})
	exitIfErr(entryLog, "registering password controller", err)

	rsaKeyReconciler := generator.NewRSAKeyReconciler(sgClient, coreClient, log.WithName("rsakey"))
	_, err = registerCtrl("rsakey", mgr, rsaKeyReconciler, &source.Kind{Type: &sgv1alpha1.RSAKey{}})
	exitIfErr(entryLog, "registering rsakey controller", err)

	sshKeyReconciler := generator.NewSSHKeyReconciler(sgClient, coreClient, log.WithName("sshkey"))
	_, err = registerCtrl("sshkey", mgr, sshKeyReconciler, &source.Kind{Type: &sgv1alpha1.SSHKey{}})
	exitIfErr(entryLog, "registering sshkey controller", err)

	secretExports := sharing.NewSecretExports(log.WithName("secretexports"))

	go func() {
		time.Sleep(10)
		{
			secretExportReconciler := sharing.NewSecretExportReconciler(
				mgr.GetClient(), secretExports, log.WithName("secexp"))

			err := secretExportReconciler.WarmUp(context.Background())
			exitIfErr(entryLog, "warmingup secexp controller", err)

			err = registerCtrlMinimal("secexp", mgr, secretExportReconciler)
			exitIfErr(entryLog, "registering secexp controller", err)
		}

		{
			secretRequestReconciler := sharing.NewSecretRequestReconciler(
				mgr.GetClient(), log.WithName("secreq"))
			err := registerCtrlMinimal("secreq", mgr, secretRequestReconciler)
			exitIfErr(entryLog, "registering secreq controller", err)
		}
	}()

	// Start after warming up secret exports
	secretReconciler := sharing.NewSecretReconciler(
		mgr.GetClient(), secretExports, log.WithName("secret"))
	err = registerCtrlMinimal("secret", mgr, secretReconciler)
	exitIfErr(entryLog, "registering secret controller", err)

	entryLog.Info("starting manager")

	err = mgr.Start(signals.SetupSignalHandler())
	exitIfErr(entryLog, "unable to run manager", err)
}

func registerCtrl(desc string, mgr manager.Manager,
	reconciler reconcile.Reconciler, src source.Source) (controller.Controller, error) {

	ctrlOpts := controller.Options{
		Reconciler: reconciler,
		// Default MaxConcurrentReconciles is 1. Keeping at that
		// since we are not doing anything that we need to parallelize for.
	}

	ctrl, err := controller.New("sg-"+desc, mgr, ctrlOpts)
	if err != nil {
		return ctrl, fmt.Errorf("unable to set up secretgen-controller-%s: %s", desc, err)
	}

	err = ctrl.Watch(src, &handler.EnqueueRequestForObject{})
	if err != nil {
		return ctrl, fmt.Errorf("unable to watch %s: %s", desc, err)
	}

	return ctrl, nil
}

type reconcilerWithWatches interface {
	reconcile.Reconciler
	AttachWatches(controller controller.Controller) error
}

func registerCtrlMinimal(desc string, mgr manager.Manager,
	reconciler reconcilerWithWatches) error {

	ctrlOpts := controller.Options{
		Reconciler: reconciler,
		// Default MaxConcurrentReconciles is 1. Keeping at that
		// since we are not doing anything that we need to parallelize for.
	}

	ctrl, err := controller.New("sg-"+desc, mgr, ctrlOpts)
	if err != nil {
		return fmt.Errorf("unable to set up secretgen-controller-%s: %s", desc, err)
	}

	err = reconciler.AttachWatches(ctrl)
	if err != nil {
		return fmt.Errorf("unable to attaches watches: %s", err)
	}

	return nil
}

func exitIfErr(entryLog logr.Logger, desc string, err error) {
	if err != nil {
		entryLog.Error(err, desc)
		os.Exit(1)
	}
}
