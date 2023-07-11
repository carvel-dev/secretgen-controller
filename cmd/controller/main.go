// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package main

// Based on https://github.com/kubernetes-sigs/controller-runtime/blob/8f633b179e1c704a6e40440b528252f147a3362a/examples/builtins/main.go

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	sgv1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen/v1alpha1"
	sg2v1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen2/v1alpha1"
	sgclient "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/client/clientset/versioned"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/generator"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/satoken"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/sharing"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/tracker"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/ratelimiter"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	// Version of secretgen-controller is set via ldflags at build-time from most recent git tag
	Version = "develop"

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
	sg2v1alpha1.AddToScheme(scheme.Scheme)

	mgr, err := manager.New(restConfig, manager.Options{Namespace: ctrlNamespace})
	exitIfErr(entryLog, "unable to set up controller manager", err)

	entryLog.Info("setting up controllers")

	coreClient, err := kubernetes.NewForConfig(restConfig)
	exitIfErr(entryLog, "building core client", err)

	sgClient, err := sgclient.NewForConfig(restConfig)
	exitIfErr(entryLog, "building secretgen client", err)

	certReconciler := generator.NewCertificateReconciler(sgClient, coreClient, log.WithName("cert"))
	exitIfErr(entryLog, "registering", registerCtrl("cert", mgr, certReconciler))

	passwordReconciler := generator.NewPasswordReconciler(sgClient, coreClient, log.WithName("password"))
	exitIfErr(entryLog, "registering", registerCtrl("password", mgr, passwordReconciler))

	rsaKeyReconciler := generator.NewRSAKeyReconciler(sgClient, coreClient, log.WithName("rsakey"))
	exitIfErr(entryLog, "registering", registerCtrl("rsakey", mgr, rsaKeyReconciler))

	sshKeyReconciler := generator.NewSSHKeyReconciler(sgClient, coreClient, log.WithName("sshkey"))
	exitIfErr(entryLog, "registering", registerCtrl("sshkey", mgr, sshKeyReconciler))

	saLoader := generator.NewServiceAccountLoader(satoken.NewManager(coreClient, log.WithName("template")))

	// Set SecretTemplate's maximum exponential to reduce reconcile time for inputresource errors
	rateLimiter := workqueue.NewItemExponentialFailureRateLimiter(100*time.Millisecond, 120*time.Second)
	secretTemplateReconciler := generator.NewSecretTemplateReconciler(mgr.GetClient(), saLoader, tracker.NewTracker(), log.WithName("template"))
	exitIfErr(entryLog, "registering", registerCtrlWithRateLimiter("template", mgr, secretTemplateReconciler, rateLimiter))

	{
		secretExports := sharing.NewSecretExportsWarmedUp(
			sharing.NewSecretExports(mgr.GetClient(), log.WithName("secretexports")))

		secretExportReconciler := sharing.NewSecretExportReconciler(
			mgr.GetClient(), secretExports, log.WithName("secexp"))
		secretExports.WarmUpFunc = secretExportReconciler.WarmUp
		exitIfErr(entryLog, "registering", registerCtrl("secexp", mgr, secretExportReconciler))

		secretImportReconciler := sharing.NewSecretImportReconciler(
			mgr.GetClient(), secretExports, log.WithName("secimp"))
		exitIfErr(entryLog, "registering", registerCtrl("secimp", mgr, secretImportReconciler))

		secretReconciler := sharing.NewSecretReconciler(
			mgr.GetClient(), secretExports, log.WithName("secret"))
		exitIfErr(entryLog, "registering", registerCtrl("secret", mgr, secretReconciler))
	}

	entryLog.Info("starting manager")

	err = mgr.Start(signals.SetupSignalHandler())
	exitIfErr(entryLog, "unable to run manager", err)
}

type reconcilerWithWatches interface {
	reconcile.Reconciler
	AttachWatches(controller.Controller) error
}

func registerCtrl(desc string, mgr manager.Manager, reconciler reconcilerWithWatches) error {
	return registerCtrlWithRateLimiter(desc, mgr, reconciler, workqueue.DefaultControllerRateLimiter())
}

func registerCtrlWithRateLimiter(desc string, mgr manager.Manager, reconciler reconcilerWithWatches, ratelimiter ratelimiter.RateLimiter) error {
	ctrlName := "sg-" + desc

	ctrlOpts := controller.Options{
		Reconciler: reconciler,
		// Default MaxConcurrentReconciles is 1. Keeping at that
		// since we are not doing anything that we need to parallelize for.

		RateLimiter: ratelimiter,
	}

	ctrl, err := controller.New(ctrlName, mgr, ctrlOpts)
	if err != nil {
		return fmt.Errorf("%s: unable to set up: %s", ctrlName, err)
	}

	err = reconciler.AttachWatches(ctrl)
	if err != nil {
		return fmt.Errorf("%s: unable to attaches watches: %s", ctrlName, err)
	}

	return nil
}

func exitIfErr(entryLog logr.Logger, desc string, err error) {
	if err != nil {
		entryLog.Error(err, desc)
		os.Exit(1)
	}
}
