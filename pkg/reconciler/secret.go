package reconciler

import (
	"fmt"

	sgv1alpha1 "github.com/k14s/secretgen-controller/pkg/apis/secretgen/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Secret struct {
	secret *corev1.Secret
	values map[string][]byte
}

func NewSecret(owner metav1.Object, values map[string][]byte) *Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        owner.GetName(),
			Namespace:   owner.GetNamespace(),
			Labels:      owner.GetLabels(),
			Annotations: owner.GetAnnotations(),
		},
	}

	controllerutil.SetControllerReference(owner, secret, scheme.Scheme)

	return &Secret{secret, values}
}

func (p *Secret) AsSecret() *corev1.Secret { return p.secret }

func (p *Secret) ApplyTemplates(defaultTpl sgv1alpha1.SecretTemplate,
	customTpl *sgv1alpha1.SecretTemplate) error {

	err := p.ApplyTemplate(defaultTpl)
	if err != nil {
		return err
	}

	if customTpl != nil {
		err := p.ApplyTemplate(*customTpl)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Secret) ApplyTemplate(template sgv1alpha1.SecretTemplate) error {
	if len(template.Metadata.Annotations) > 0 {
		if p.secret.Annotations == nil {
			p.secret.Annotations = map[string]string{}
		}
		for k, v := range template.Metadata.Annotations {
			p.secret.Annotations[k] = v
		}
	}

	if len(template.Metadata.Labels) > 0 {
		if p.secret.Labels == nil {
			p.secret.Labels = map[string]string{}
		}
		for k, v := range template.Metadata.Labels {
			p.secret.Labels[k] = v
		}
	}

	if len(template.Type) > 0 {
		p.secret.Type = template.Type
	}

	if len(template.Data) > 0 {
		newData := map[string][]byte{}
		for dataKey, valKey := range template.Data {
			val, found := p.values[valKey]
			if !found {
				return fmt.Errorf("Expected value for key '%s' to be found, but was not", valKey)
			}
			newData[dataKey] = val
		}
		p.secret.Data = newData
	}

	return nil
}

func (p *Secret) ApplySecret(otherSecret *corev1.Secret) {
	if len(otherSecret.Annotations) > 0 {
		if p.secret.Annotations == nil {
			p.secret.Annotations = map[string]string{}
		}
		for k, v := range otherSecret.Annotations {
			// Do not override existing annotations
			if _, found := p.secret.Annotations[k]; !found {
				p.secret.Annotations[k] = v
			}
		}
	}

	if len(otherSecret.Labels) > 0 {
		if p.secret.Labels == nil {
			p.secret.Labels = map[string]string{}
		}
		for k, v := range otherSecret.Labels {
			// Do not override existing labels
			if _, found := p.secret.Labels[k]; !found {
				p.secret.Labels[k] = v
			}
		}
	}

	if len(otherSecret.Type) > 0 {
		p.secret.Type = otherSecret.Type
	}

	if len(otherSecret.Data) > 0 {
		newData := map[string][]byte{}
		for k, v := range otherSecret.Data {
			newData[k] = v
		}
		p.secret.Data = newData
	}
}

func (p *Secret) AssociteExistingSecret(otherSecret *corev1.Secret) {
	p.secret.UID = otherSecret.UID
	p.secret.ResourceVersion = otherSecret.ResourceVersion
}
