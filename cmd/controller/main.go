package main

// Based on https://github.com/kubernetes-sigs/controller-runtime/blob/8f633b179e1c704a6e40440b528252f147a3362a/examples/builtins/main.go

import (
	"flag"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	sgv1alpha1 "github.com/k14s/secretgen-controller/pkg/apis/secretgen/v1alpha1"
	sgclient "github.com/k14s/secretgen-controller/pkg/client/clientset/versioned"
	"github.com/k14s/secretgen-controller/pkg/reconciler"
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

var (
	log             = logf.Log.WithName("secretgen-controller")
	ctrlConcurrency = 10
	ctrlNamespace   = ""
)

func main() {
	flag.IntVar(&ctrlConcurrency, "concurrency", 10, "Max concurrent reconciles")
	flag.StringVar(&ctrlNamespace, "namespace", "", "Namespace to watch")
	flag.Parse()

	logf.SetLogger(zap.Logger(false))
	entryLog := log.WithName("entrypoint")

	entryLog.Info("setting up manager")
	restConfig := config.GetConfigOrDie()

	// Register sg API types so they can be watched
	sgv1alpha1.AddToScheme(scheme.Scheme)

	mgr, err := manager.New(restConfig, manager.Options{Namespace: ctrlNamespace})
	exitIfErr(entryLog, "unable to set up controller manager", err)

	entryLog.Info("Setting up controllers")

	coreClient, err := kubernetes.NewForConfig(restConfig)
	exitIfErr(entryLog, "building core client", err)

	sgClient, err := sgclient.NewForConfig(restConfig)
	exitIfErr(entryLog, "building secretgen client", err)

	certReconciler := reconciler.NewCertificateReconciler(sgClient, coreClient, log.WithName("cert"))
	_, err = registerCtrl("cert", mgr, certReconciler, &source.Kind{Type: &sgv1alpha1.Certificate{}})
	exitIfErr(entryLog, "registering certificate controller", err)

	passwordReconciler := reconciler.NewPasswordReconciler(sgClient, coreClient, log.WithName("password"))
	_, err = registerCtrl("password", mgr, passwordReconciler, &source.Kind{Type: &sgv1alpha1.Password{}})
	exitIfErr(entryLog, "registering password controller", err)

	rsaKeyReconciler := reconciler.NewRSAKeyReconciler(sgClient, coreClient, log.WithName("rsakey"))
	_, err = registerCtrl("rsakey", mgr, rsaKeyReconciler, &source.Kind{Type: &sgv1alpha1.RSAKey{}})
	exitIfErr(entryLog, "registering rsakey controller", err)

	sshKeyReconciler := reconciler.NewSSHKeyReconciler(sgClient, coreClient, log.WithName("sshkey"))
	_, err = registerCtrl("sshkey", mgr, sshKeyReconciler, &source.Kind{Type: &sgv1alpha1.SSHKey{}})
	exitIfErr(entryLog, "registering sshkey controller", err)

	entryLog.Info("starting manager")

	err = mgr.Start(signals.SetupSignalHandler())
	exitIfErr(entryLog, "unable to run manager", err)
}

func registerCtrl(desc string, mgr manager.Manager,
	reconciler reconcile.Reconciler, src source.Source) (controller.Controller, error) {

	ctrlOpts := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: ctrlConcurrency,
	}

	ctrl, err := controller.New("secretgen-controller-"+desc, mgr, ctrlOpts)
	if err != nil {
		return ctrl, fmt.Errorf("unable to set up secretgen-controller-%s: %s", desc, err)
	}

	err = ctrl.Watch(src, &handler.EnqueueRequestForObject{})
	if err != nil {
		return ctrl, fmt.Errorf("unable to watch %s: %s", desc, err)
	}

	return ctrl, nil
}

func exitIfErr(entryLog logr.Logger, desc string, err error) {
	if err != nil {
		entryLog.Error(err, desc)
		os.Exit(1)
	}
}
