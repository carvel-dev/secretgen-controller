// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package sharing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"sync"

	"github.com/go-logr/logr"
	sg2v1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen2/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/jsonpath"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// WeightAnnKey allows to control which secrets are preferred to others
	// during fulfillment of secret requests. It's especially handy for
	// controlling how multiple image pull secrets are merged together.
	WeightAnnKey = "secretgen.carvel.dev/weight"
)

// NamespaceWildcardExclusionCheck is a function that takes the name of a namespace
// and returns whether that ns is excluded from wildcard matches
type NamespaceWildcardExclusionCheck func(string) bool

// SecretExportsProvider provides a way to record and
// later query secrets based on a given criteria.
type SecretExportsProvider interface {
	Export(*sg2v1alpha1.SecretExport, *corev1.Secret)
	Unexport(*sg2v1alpha1.SecretExport)
	MatchedSecretsForImport(SecretMatcher, NamespaceWildcardExclusionCheck) []*corev1.Secret
}

// SecretExports is an in-memory cache of exported secrets.
// It can be asked to return secrets that match specific criteria for importing.
// (SecretExports is used by SecretExportReconciler to export/unexport secrets;
// SecretExports is used by SecretReconciler to determine imported secrets.)
type SecretExports struct {
	log       logr.Logger
	k8sReader K8sReader

	exportedSecretsLock sync.RWMutex
	exportedSecrets     map[string]exportedSecret
}

// K8sReader is an interface for reading Kubernetes resources.
type K8sReader interface {
	Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error
}

var _ SecretExportsProvider = &SecretExports{}

// NewSecretExports constructs new SecretExports cache.
func NewSecretExports(k8sReader K8sReader, log logr.Logger) *SecretExports {
	return &SecretExports{log: log, k8sReader: k8sReader, exportedSecrets: map[string]exportedSecret{}}
}

// Export adds the in-memory representation (cached)
// of both the SecretExport and underlying Secret.
func (se *SecretExports) Export(export *sg2v1alpha1.SecretExport, secret *corev1.Secret) {
	if secret == nil {
		panic("Internal inconsistency: expected non-nil secret")
	}
	exportedSec := newExportedSecret(export, secret)

	se.exportedSecretsLock.Lock()
	defer se.exportedSecretsLock.Unlock()

	se.exportedSecrets[exportedSec.Key()] = exportedSec
}

// Unexport deletes the in-memory representation (cached)
// of both the SecretExport and underlying Secret.
func (se *SecretExports) Unexport(export *sg2v1alpha1.SecretExport) {
	exportedSec := newExportedSecret(export, nil)

	se.exportedSecretsLock.Lock()
	defer se.exportedSecretsLock.Unlock()

	delete(se.exportedSecrets, exportedSec.Key())
}

// SecretMatcher allows to specify criteria for matching exported secrets.
type SecretMatcher struct {
	FromName      string
	FromNamespace string

	ToNamespace string

	Subject    string
	SecretType corev1.SecretType

	Ctx context.Context
}

// NamespacesMatcher allows to specify criteria for matching exported secrets based on namespaces fields.
type NamespacesMatcher struct {
	Selectors []sg2v1alpha1.SelectorMatchField
}

// MatchNamespace returns true if the given SecretMatcher matches one of the Selectors.
func (nm NamespacesMatcher) MatchNamespace(matcher SecretMatcher, log logr.Logger, k8sReader K8sReader) bool {

	selectors := nm.Selectors
	nsName := matcher.ToNamespace
	query := types.NamespacedName{
		Name: nsName,
	}
	namespace := corev1.Namespace{}
	err := k8sReader.Get(matcher.Ctx, query, &namespace)

	jsonNsString, _ := json.Marshal(namespace)
	var jsonNsObject interface{}
	json.Unmarshal(jsonNsString, &jsonNsObject)

	if err != nil {
		log.Error(err, fmt.Sprintf("failed to get namespace %s", nsName))
	}
	for _, s := range selectors {
		jp := jsonpath.New("jsonpath")

		jsonPathKey := "{." + s.Key + "}"
		err := jp.Parse(jsonPathKey)
		if err != nil {
			log.Error(err, fmt.Sprintf("invalid jsonpath: %s", jsonPathKey))
			return false
		}
		var valueBuffer bytes.Buffer
		err = jp.Execute(&valueBuffer, jsonNsObject)
		value := valueBuffer.String()

		switch s.Operator {
		case sg2v1alpha1.SelectorOperatorIn:
			found := false
			for _, svalue := range s.Values {
				if svalue == value {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		case sg2v1alpha1.SelectorOperatorNotIn:
			for _, svalue := range s.Values {
				if svalue == value {
					return false
				}
			}
		case sg2v1alpha1.SelectorOperatorExists:
			if value == "" {
				return false
			}
		case sg2v1alpha1.SelectorOperatorDoesNotExist:
			if value != "" {
				return false
			}
		}
	}
	return true
}

// MatchedSecretsForImport filters secrets export cache by the given criteria.
// Returned order (last in the array is most specific):
//   - secret with highest weight? (default weight=0), or
//   - secret within the same namespace
//   - secret with specific namespace
//   - secret with wildcard namespace match
//   - secret within other namespaces
//   - secret with specific namespace
//   - secret with wildcard namespace match
//     (in all cases fallback to secret namespace/name sort)
func (se *SecretExports) MatchedSecretsForImport(matcher SecretMatcher, nsIsExcludedFromWildcard NamespaceWildcardExclusionCheck) []*corev1.Secret {
	se.exportedSecretsLock.RLock()
	defer se.exportedSecretsLock.RUnlock()

	var matched []exportedSecret

	for _, exportedSec := range se.exportedSecrets {
		if exportedSec.Matches(matcher, nsIsExcludedFromWildcard, se.log, se.k8sReader) {
			matched = append(matched, exportedSec)
		}
	}

	sort.Slice(matched, func(i, j int) bool {
		// j and i are flipped to do a reverse sort
		return matched[j].SortKey(matcher.ToNamespace).Less(matched[i].SortKey(matcher.ToNamespace))
	})

	var result []*corev1.Secret
	for _, exportedSec := range matched {
		result = append(result, exportedSec.Secret())
	}

	return result
}

// exportedSecret is used for keeping track export->secret pair.
type exportedSecret struct {
	export            *sg2v1alpha1.SecretExport
	secret            *corev1.Secret
	namespacesMatcher NamespacesMatcher
}

func newExportedSecret(export *sg2v1alpha1.SecretExport, secret *corev1.Secret) exportedSecret {
	if export == nil {
		panic("Internal inconsistency: nil export")
	}
	if export.Namespace == "" {
		panic("Internal inconsistency: missing export namespace")
	}
	if export.Name == "" {
		panic("Internal inconsistency: missing export name")
	}
	if secret != nil {
		if export.Namespace != secret.Namespace || export.Name != secret.Name {
			panic("Internal inconsistency: export and secret names do not match")
		}
		secret = secret.DeepCopy()
	}

	namespacesMatcher := NamespacesMatcher{Selectors: export.Spec.ToNamespacesSelector}

	return exportedSecret{export.DeepCopy(), secret, namespacesMatcher}
}

func (es exportedSecret) Key() string {
	return es.export.Namespace + "/" + es.export.Name
}

func (es exportedSecret) Secret() *corev1.Secret {
	return es.secret.DeepCopy()
}

func (es exportedSecret) Matches(matcher SecretMatcher, nsIsExcludedFromWildcard NamespaceWildcardExclusionCheck, log logr.Logger, k8sReader K8sReader) bool {

	if matcher.Subject != "" {
		// TODO we currently do not match by subject
		log.Info("Warning: Matcher has empty subject and will never match any secret")
		return false
	}

	if len(matcher.SecretType) > 0 {
		if matcher.SecretType != es.secret.Type {
			return false
		}
	}
	if len(matcher.FromName) > 0 {
		if matcher.FromName != es.secret.Name {
			return false
		}
	}

	if len(matcher.FromNamespace) > 0 {
		if matcher.FromNamespace != es.secret.Namespace {
			return false
		}
	}

	namespacesMatcher := es.namespacesMatcher

	if es.matchesNamespace(matcher.ToNamespace, nsIsExcludedFromWildcard) {
		return true
	}

	if len(namespacesMatcher.Selectors) > 0 {
		return namespacesMatcher.MatchNamespace(matcher, log, k8sReader)
	}

	return false
}

func (es exportedSecret) matchesNamespace(nsToMatch string, nsIsExcludedFromWildcard NamespaceWildcardExclusionCheck) bool {
	for _, ns := range es.export.StaticToNamespaces() {
		if ns == nsToMatch {
			return true
		}
		if ns == sg2v1alpha1.AllNamespaces && !nsIsExcludedFromWildcard(nsToMatch) {
			return true
		}
	}
	return false
}

func (es exportedSecret) SortKey(toNs string) exportedSecretSortKey {
	var weight float64 // default weight is 0.0
	if val, found := es.export.Annotations[WeightAnnKey]; found {
		if typedVal, err := strconv.ParseFloat(val, 64); err == nil { // Ignore invalid weights
			weight = typedVal
		}
	}

	var matchesToNsExactly bool
	for _, ns := range es.export.StaticToNamespaces() {
		if ns == toNs {
			matchesToNsExactly = true
			break
		}
	}

	return exportedSecretSortKey{
		Weight:             weight,
		WithinToNs:         es.secret.Namespace == toNs,
		MatchesToNsExactly: matchesToNsExactly,
		SecretNsName:       fmt.Sprintf("%s/%s", es.secret.Namespace, es.secret.Name),
	}
}

type exportedSecretSortKey struct {
	Weight             float64
	WithinToNs         bool
	MatchesToNsExactly bool // or by wildcard
	SecretNsName       string
}

func (k exportedSecretSortKey) Less(otherKey exportedSecretSortKey) bool {
	// Check weights
	if k.Weight > otherKey.Weight {
		return true
	}
	if k.Weight < otherKey.Weight {
		return false
	}

	// Check same dst namespace
	if k.WithinToNs && !otherKey.WithinToNs {
		return true
	}
	if !k.WithinToNs && otherKey.WithinToNs {
		return false
	}

	// Check ns name exact match
	if k.MatchesToNsExactly && !otherKey.MatchesToNsExactly {
		return true
	}
	if !k.MatchesToNsExactly && otherKey.MatchesToNsExactly {
		return false
	}

	// Lastly just by ns/name
	return k.SecretNsName > otherKey.SecretNsName
}
