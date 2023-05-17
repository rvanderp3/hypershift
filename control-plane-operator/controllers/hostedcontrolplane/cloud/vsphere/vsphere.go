package vsphere

import (
	"bytes"
	"fmt"

	hyperv1 "github.com/openshift/hypershift/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

const (
	CloudConfigKey = "vsphere.conf"
	Provider       = "vsphere"
)

func printIfNotEmpty(buf *bytes.Buffer, k, v string) {
	if v != "" {
		fmt.Fprintf(buf, "%s = %q\n", k, v)
	}
}

// ReconcileCloudConfig reconciles as expected by Nodes Kubelet.
func ReconcileCloudConfig(cm *corev1.ConfigMap, hcp *hyperv1.HostedControlPlane) error {
	cfg := vsphereConfigWithoutCredentials(hcp, hcp.Spec.Platform.VSphere.SecretName)
	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	cm.Data[CloudConfigKey] = cfg
	return nil
}

func vsphereConfigWithoutCredentials(hcp *hyperv1.HostedControlPlane, credentialsSecret string) string {
	buf := new(bytes.Buffer)
	p := hcp.Spec.Platform.VSphere

	fmt.Fprintln(buf, "[Global]")
	printIfNotEmpty(buf, "secret-name", "vsphere-creds")
	printIfNotEmpty(buf, "secret-namespace", "kube-system")
	printIfNotEmpty(buf, "insecure-flag", "1")
	fmt.Fprintln(buf, "")
	fmt.Fprintln(buf, "[Workspace]")
	printIfNotEmpty(buf, "server", p.VCenter)
	printIfNotEmpty(buf, "datacenter", p.Datacenter)
	printIfNotEmpty(buf, "default-datastore", p.DefaultDatastore)
	printIfNotEmpty(buf, "folder", p.Folder)
	printIfNotEmpty(buf, "resourcepool-path", p.ResourcePool)
	fmt.Fprintln(buf, "")
	fmt.Fprintf(buf, "[VirtualCenter %q]\n", p.VCenter)
	printIfNotEmpty(buf, "datacenters", p.Datacenter)
	return buf.String()
}
