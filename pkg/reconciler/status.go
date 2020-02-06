package reconciler

import (
	"fmt"

	sgv1alpha1 "github.com/k14s/secretgen-controller/pkg/apis/secretgen/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Status struct {
	s          sgv1alpha1.GenericStatus
	updateFunc func(sgv1alpha1.GenericStatus)
}

func (s *Status) Result() sgv1alpha1.GenericStatus { return s.s }

func (s *Status) SetReconciling(meta metav1.ObjectMeta) {
	s.markObservedLatest(meta)
	s.removeAllConditions()

	s.s.Conditions = append(s.s.Conditions, sgv1alpha1.Condition{
		Type:   sgv1alpha1.Reconciling,
		Status: corev1.ConditionTrue,
	})

	s.s.FriendlyDescription = "Reconciling"

	s.updateFunc(s.s)
}

func (s *Status) SetReconcileCompleted(err error) {
	s.removeAllConditions()

	if err != nil {
		s.s.Conditions = append(s.s.Conditions, sgv1alpha1.Condition{
			Type:    sgv1alpha1.ReconcileFailed,
			Status:  corev1.ConditionTrue,
			Message: err.Error(),
		})
		s.s.FriendlyDescription = fmt.Sprintf("Reconcile failed: %s", err.Error())
	} else {
		s.s.Conditions = append(s.s.Conditions, sgv1alpha1.Condition{
			Type:    sgv1alpha1.ReconcileSucceeded,
			Status:  corev1.ConditionTrue,
			Message: "",
		})
		s.s.FriendlyDescription = "Reconcile succeeded"
	}

	s.updateFunc(s.s)
}

func (s *Status) WithReconcileCompleted(result reconcile.Result, err error) (reconcile.Result, error) {
	s.SetReconcileCompleted(err)
	return result, err
}

func (s *Status) markObservedLatest(meta metav1.ObjectMeta) {
	s.s.ObservedGeneration = meta.Generation
}

func (s *Status) removeAllConditions() {
	s.s.Conditions = nil
}
