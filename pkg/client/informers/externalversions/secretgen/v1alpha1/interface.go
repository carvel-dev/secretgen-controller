// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	internalinterfaces "carvel.dev/secretgen-controller/pkg/client/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// Certificates returns a CertificateInformer.
	Certificates() CertificateInformer
	// Passwords returns a PasswordInformer.
	Passwords() PasswordInformer
	// RSAKeys returns a RSAKeyInformer.
	RSAKeys() RSAKeyInformer
	// SSHKeys returns a SSHKeyInformer.
	SSHKeys() SSHKeyInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// Certificates returns a CertificateInformer.
func (v *version) Certificates() CertificateInformer {
	return &certificateInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Passwords returns a PasswordInformer.
func (v *version) Passwords() PasswordInformer {
	return &passwordInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// RSAKeys returns a RSAKeyInformer.
func (v *version) RSAKeys() RSAKeyInformer {
	return &rSAKeyInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// SSHKeys returns a SSHKeyInformer.
func (v *version) SSHKeys() SSHKeyInformer {
	return &sSHKeyInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
