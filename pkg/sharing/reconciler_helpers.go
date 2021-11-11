// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package sharing

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// makeNamespaceExclusionCheck returns a function that uses reconciler-level tools (k8s client, logger, context) to
// check the presence of a namespace annotation that we mostly only care about in the inner workings of SecretExport.
func makeNamespaceExclusionCheck(ctx context.Context, cli client.Client, log logr.Logger) NamespaceExclusionCheck {
	return func(nsName string) bool {
		query := types.NamespacedName{
			Name: nsName,
		}
		namespace := corev1.Namespace{}
		err := cli.Get(ctx, query, &namespace)
		if err != nil {
			log.Error(err, "Called to check annotation on a namespace but couldn't find:", "namespace", nsName)
			return false
		}
		_, excluded := namespace.Annotations["secretgen.carvel.dev/excluded-from-wildcard-matching"]
		return excluded
	}
}
