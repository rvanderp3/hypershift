package vsphere

import (
	"context"
	"github.com/openshift/hypershift/api/v1beta1"
	"os"
	"os/signal"
	"syscall"

	hyperv1 "github.com/openshift/hypershift/api/v1beta1"
	"github.com/openshift/hypershift/cmd/log"
	"github.com/openshift/hypershift/cmd/nodepool/core"
	"github.com/spf13/cobra"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewCreateCommand(coreOpts *core.CreateNodePoolOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "vsphere",
		Short:        "Creates a vSphere nodepool",
		SilenceUsage: true,
	}
	o := &VSpherePlatformCreateOptions{
		diskSizeGB:     120,
		cpus:           4,
		memoryMB:       16384,
		coresPerSocket: 1,
	}
	cmd.Flags().StringVar(&o.template, "template", o.template, "The name of the VM or template which is cloned to create new nodes")
	cmd.Flags().Int32Var(&o.diskSizeGB, "disk-size", o.diskSizeGB, "The size of the root disk for machines in the NodePool (minimum 16)")
	cmd.Flags().Int32Var(&o.cpus, "cpus", o.cpus, "The number of vCPUs allocated to a node")
	cmd.Flags().Int32Var(&o.coresPerSocket, "cores-per-socket", o.coresPerSocket, "Defines the topology of cores per socket to the node")
	cmd.Flags().Int32Var(&o.memoryMB, "memory", o.memoryMB, "The amount of memory in MB allocated to the node")
	cmd.Flags().StringVar(&o.resourcePool, "resource-pool", o.resourcePool, "The full path to the resource pool where nodes are to be deployed")
	cmd.Flags().StringVar(&o.cluster, "cluster", o.cluster, "The cluster where nodes are to be deployed")
	cmd.Flags().StringVar(&o.defaultDatastore, "datastore", o.defaultDatastore, "The datastore where nodes are to be deployed")
	cmd.Flags().StringVar(&o.folder, "folder", o.folder, "The VM folder where nodes are to be deployed")
	cmd.Flags().StringVar(&o.network, "network", o.network, "The port group attached to the node to be deployed")
	cmd.Flags().StringVar(&o.datacenter, "datacenter", o.network, "The datacenter attached to the node to be deployed")
	cmd.Flags().StringVar(&o.vcenter, "vcenter", o.vcenter, "The datacenter attached to the node to be deployed")
	cmd.MarkFlagRequired("template")
	cmd.MarkFlagRequired("network")
	cmd.MarkFlagRequired("datacenter")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT)
		go func() {
			<-sigs
			cancel()
		}()

		if err := coreOpts.CreateNodePool(ctx, o); err != nil {
			log.Log.Error(err, "Failed to create nodepool")
			os.Exit(1)
		}
	}

	return cmd
}

type VSpherePlatformCreateOptions struct {
	// Template is the name of the VM or template which is cloned to create new nodes
	template string `json:"template"`

	// +kubebuilder:default:=120
	// +kubebuilder:validation:Minimum=16
	// +optional
	diskSizeGB int32 `json:"diskSizeGB,omitempty"`

	// cpus is the number of vCPUs allocated to a node
	// +kubebuilder:default:=4
	// +kubebuilder:validation:Minimum=2
	// +optional
	cpus int32 `json:"cpus,omitempty"`

	// coresPerSocket defines the topology of cores per socket to the node
	// +kubebuilder:default:=1
	// +kubebuilder:validation:Minimum=1
	// +optional
	coresPerSocket int32 `json:"coresPerSocket,omitempty"`

	// memoryMB defines the amount of memory allocated to the node
	// +kubebuilder:default:=16384
	// +kubebuilder:validation:Minimum=8192
	// +optional
	memoryMB int32 `json:"memoryMB,omitempty"`

	// datacenter is the name of the datacenter to use in the vCenter.
	datacenter string `json:"datacenter"`

	// defaultDatastore is the default datastore to use for provisioning volumes.
	defaultDatastore string `json:"defaultDatastore"`

	// folder is the absolute path of the folder that will be used and/or created for
	// virtual machines. The absolute path is of the form /<datacenter>/vm/<folder>/<subfolder>.
	folder string `json:"folder,omitempty"`

	// cluster is the name of the cluster virtual machines will be cloned into.
	cluster string `json:"cluster,omitempty"`

	// resourcePool is the absolute path of the resource pool where virtual machines will be
	// created. The absolute path is of the form /<datacenter>/host/<cluster>/Resources/<resourcepool>.
	resourcePool string `json:"resourcePool,omitempty"`

	// network specifies the name of the network to be used by the cluster.
	network string `json:"network,omitempty"`

	// vcenter specifies the hostname of the vcenter
	vcenter string `json:"vcenter,omitempty"`
}

func (o VSpherePlatformCreateOptions) UpdateNodePool(ctx context.Context, nodePool *v1beta1.NodePool, hcluster *v1beta1.HostedCluster, client crclient.Client) error {
	nodePool.Spec.Platform.VSphere = &hyperv1.VSphereNodePoolPlatform{
		Template:         o.template,
		Network:          o.network,
		ResourcePool:     o.resourcePool,
		Folder:           o.folder,
		Cluster:          o.cluster,
		DefaultDatastore: o.defaultDatastore,
		Datacenter:       o.datacenter,
		MemoryMB:         o.memoryMB,
		CoresPerSocket:   o.coresPerSocket,
		Cpus:             o.cpus,
		DiskSizeGB:       o.diskSizeGB,
	}
	nodePool.Spec.Platform.Type = hyperv1.VSpherePlatform
	return nil
}

func (V VSpherePlatformCreateOptions) Type() v1beta1.PlatformType {
	return hyperv1.VSpherePlatform
}
