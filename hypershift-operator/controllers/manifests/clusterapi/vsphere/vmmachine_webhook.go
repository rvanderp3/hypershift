package vsphere

import (
	"context"

	"github.com/davecgh/go-spew/spew"
	"sigs.k8s.io/cluster-api-provider-vsphere/apis/v1beta1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// Webhook implements a mutating webhook for HostedCluster.
type Webhook struct{}

type vSphereVM v1beta1.VSphereVM

func (r *vSphereVM) Default(ctx context.Context, obj runtime.Object) error {
	return nil
}

func (r *vSphereVM) DeepCopyObject() runtime.Object {
	return r
}

var _ webhook.CustomDefaulter = &vSphereVM{}

// SetupWebhookWithManager sets up HostedCluster webhooks.
func SetupWebhookWithManager(mgr ctrl.Manager) error {
	spew.Printf("installing vsphere webhook")
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1beta1.VSphereVM{}).
		WithDefaulter(&vSphereVM{}).
		Complete()
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type.
func (webhook *Webhook) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	vm := obj.(*v1beta1.VSphereVM)
	spew.Dump(vm)
	vm.Spec.CustomVMXKeys["guestinfo.hostname"] = vm.Name
	return nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type.
func (webhook *Webhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	vm := newObj.(*v1beta1.VSphereVM)
	spew.Dump(vm)
	vm.Spec.CustomVMXKeys["guestinfo.hostname"] = vm.Name
	return nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type.
func (webhook *Webhook) ValidateDelete(_ context.Context, obj runtime.Object) error {
	return nil
}
