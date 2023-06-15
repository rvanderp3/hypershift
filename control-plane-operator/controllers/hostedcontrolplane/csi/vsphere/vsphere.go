package kubevirt

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"

	hyperv1 "github.com/openshift/hypershift/api/v1beta1"
	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/imageprovider"
	"github.com/openshift/hypershift/support/config"
	"github.com/openshift/hypershift/support/upsert"
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrl "sigs.k8s.io/controller-runtime"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

//go:embed files/*
var resources embed.FS

var (
	controllerDeployment = mustDeployment("08_deployment.yaml")
	/*infraRole                          = mustRole("infra_role.yaml")
	infraRoleBinding                   = mustRoleBinding("infra_rolebinding.yaml")
	tenantControllerClusterRole        = mustClusterRole("tenant_controller_clusterrole.yaml")
	tenantControllerClusterRoleBinding = mustClusterRoleBinding("tenant_controller_clusterrolebinding.yaml")

	tenantNodeClusterRole        = mustClusterRole("tenant_node_clusterrole.yaml")
	tenantNodeClusterRoleBinding = mustClusterRoleBinding("tenant_node_clusterrolebinding.yaml")

	daemonset = mustDaemonSet("daemonset.yaml")

	defaultResourceRequirements = corev1.ResourceRequirements{Requests: corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("10m"),
		corev1.ResourceMemory: resource.MustParse("50Mi"),
	}}*/
)

func mustDeployment(file string) *appsv1.Deployment {

	controllerBytes := getContentsOrDie(file)
	controller := &appsv1.Deployment{}
	if err := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(controllerBytes), 500).Decode(&controller); err != nil {
		panic(err)
	}

	return controller
}

func mustDaemonSet(file string) *appsv1.DaemonSet {
	b := getContentsOrDie(file)
	obj := &appsv1.DaemonSet{}
	if err := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(b), 500).Decode(&obj); err != nil {
		panic(err)
	}

	return obj
}

func mustClusterRole(file string) *rbacv1.ClusterRole {
	b := getContentsOrDie(file)
	obj := &rbacv1.ClusterRole{}
	if err := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(b), 500).Decode(&obj); err != nil {
		panic(err)
	}

	return obj
}

func mustClusterRoleBinding(file string) *rbacv1.ClusterRoleBinding {
	b := getContentsOrDie(file)
	obj := &rbacv1.ClusterRoleBinding{}
	if err := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(b), 500).Decode(&obj); err != nil {
		panic(err)
	}

	return obj
}

func mustRole(file string) *rbacv1.Role {
	b := getContentsOrDie(file)
	obj := &rbacv1.Role{}
	if err := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(b), 500).Decode(&obj); err != nil {
		panic(err)
	}

	return obj
}

func mustRoleBinding(file string) *rbacv1.RoleBinding {
	b := getContentsOrDie(file)
	obj := &rbacv1.RoleBinding{}
	if err := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(b), 500).Decode(&obj); err != nil {
		panic(err)
	}

	return obj
}

func getContentsOrDie(file string) []byte {
	f, err := resources.Open("files/" + file)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()
	b, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}
	return b
}

func reconcileOperatorDeployment(ctx context.Context, controller *appsv1.Deployment, hcp *hyperv1.HostedControlPlane, componentImages map[string]string) error {
	controller.Spec = *controllerDeployment.Spec.DeepCopy()
	log := ctrl.LoggerFrom(ctx)

	imageMap := map[string]string{
		"DRIVER_IMAGE":                "vsphere-csi-driver",
		"PROVISIONER_IMAGE":           "csi-external-provisioner",
		"ATTACHER_IMAGE":              "csi-external-attacher",
		"RESIZER_IMAGE":               "csi-external-resizer",
		"SNAPSHOTTER_IMAGE":           "csi-external-snapshotter",
		"NODE_DRIVER_REGISTRAR_IMAGE": "csi-node-driver-registrar",
		"LIVENESS_PROBE_IMAGE":        "csi-livenessprobe",
		"VMWARE_VSPHERE_SYNCER_IMAGE": "vsphere-csi-driver-syncer",
		"KUBE_RBAC_PROXY_IMAGE":       "kube-rbac-proxy",
		"OPERATOR_IMAGE":              "vsphere-csi-driver-operator",
	}

	templateMap := map[string]string{}

	for key, imageName := range imageMap {
		image, exists := componentImages[imageName]
		if !exists {
			return fmt.Errorf("unable to retrieve image %s from release payload", image)
		}
		templateMap[key] = image
	}
	containers := controller.Spec.Template.Spec.Containers
	if len(containers) == 0 {
		return fmt.Errorf("no containers in template")
	}

	container := containers[0]
	for idx, envVariable := range container.Env {
		image, exists := templateMap[envVariable.Name]
		if !exists {
			continue
		}
		container.Env[idx].Value = image
		log.Info(fmt.Sprintf("substituting image %s with %s", &envVariable.Name, &container.Image))
	}

	container.Image = templateMap["OPERATOR_IMAGE"]
	log.Info(fmt.Sprintf("container image %s", &container.Image))
	ownerRef := config.OwnerRefFrom(hcp)
	ownerRef.ApplyTo(controller)

	return nil
}

// ReconcileInfra reconciles the csi driver controller on the underlying infra/Mgmt cluster
// that is hosting the KubeVirt VMs.
func ReconcileInfra(client crclient.Client, hcp *hyperv1.HostedControlPlane, ctx context.Context, createOrUpdate upsert.CreateOrUpdateFN, releaseImageProvider *imageprovider.ReleaseImageProvider) error {

	deployment := appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      "vsphere-csi-driver-operator",
			Namespace: hcp.Namespace,
		},
	}

	_, err := createOrUpdate(ctx, client, &deployment, func() error {
		return reconcileOperatorDeployment(ctx, &deployment, releaseImageProvider.ComponentImages())
	})
	if err != nil {
		return err
	}

	return nil
}
