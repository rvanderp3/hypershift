package vsphere

import (
	"context"
	"encoding/json"
	"fmt"
	apifixtures "github.com/openshift/hypershift/api/fixtures"
	hyperv1 "github.com/openshift/hypershift/api/v1beta1"
	"github.com/openshift/hypershift/cmd/cluster/core"
	awsinfra "github.com/openshift/hypershift/cmd/infra/aws"
	"github.com/openshift/hypershift/cmd/util"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
)

func NewCreateCommand(opts *core.CreateOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "vsphere",
		Short:        "Creates basic functional HostedCluster resources on vSphere",
		SilenceUsage: true,
	}

	opts.VSpherePlatform = core.VSpherePlatformOptions{
		TemplateVM:        "",
		NumCPUs:           4,
		NumCoresPerSocket: 1,
		MemoryMiB:         16384,
		DiskSizeGB:        120,
	}

	cmd.Flags().StringVar(&opts.VSpherePlatform.VCenter, "vcenter", opts.VSpherePlatform.VCenter, "vCenter hostname")
	cmd.Flags().StringVar(&opts.VSpherePlatform.Datacenter, "datacenter", opts.VSpherePlatform.Datacenter, "datacenter where compute VMs will be created")
	cmd.Flags().StringVar(&opts.VSpherePlatform.Folder, "folder", opts.VSpherePlatform.Folder, "folder where compute VMs will be created")
	cmd.Flags().StringVar(&opts.VSpherePlatform.ResourcePool, "resource-pool", opts.VSpherePlatform.ResourcePool, "resource pool where compute VMs will be created")
	cmd.Flags().StringVar(&opts.VSpherePlatform.Cluster, "cluster", opts.VSpherePlatform.Cluster, "cluster where compute VMs will be created")
	cmd.Flags().StringVar(&opts.VSpherePlatform.DefaultDatastore, "default-datastore", opts.VSpherePlatform.DefaultDatastore, "datastore where compute VMs will be created")
	cmd.Flags().StringVar(&opts.VSpherePlatform.TemplateVM, "template", opts.VSpherePlatform.TemplateVM, "name of VM which will be used as a template for compute VMs")
	cmd.Flags().Int32("disk-size", opts.VSpherePlatform.DiskSizeGB, "disk size in GB of assigned to compute VMs")
	cmd.Flags().Int32("vCPUs", opts.VSpherePlatform.NumCPUs, "number of vCPUs assigned to compute VMs")
	cmd.Flags().Int32("cores-per-socket", opts.VSpherePlatform.NumCoresPerSocket, "number of cores per socket assigned to compute VMs")
	cmd.Flags().Int64("memory", opts.VSpherePlatform.MemoryMiB, "amount of memory in MiB assigned to compute VMs")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT)
		go func() {
			<-sigs
			cancel()
		}()

		if err := CreateCluster(ctx, opts); err != nil {
			opts.Log.Error(err, "Failed to create cluster")
			os.Exit(1)
		}
	}

	return cmd
}

func CreateCluster(ctx context.Context, opts *core.CreateOptions) error {
	if err := core.Validate(ctx, opts); err != nil {
		return err
	}
	return core.CreateCluster(ctx, opts, applyPlatformSpecificsValues)
}

func applyPlatformSpecificsValues(ctx context.Context, exampleOptions *apifixtures.ExampleOptions, opts *core.CreateOptions) (err error) {
	client, err := util.GetClient()
	if err != nil {
		return err
	}
	infraID := opts.InfraID

	// Load or create infrastructure for the cluster
	var infra *awsinfra.CreateInfraOutput
	if len(opts.InfrastructureJSON) > 0 {
		rawInfra, err := ioutil.ReadFile(opts.InfrastructureJSON)
		if err != nil {
			return fmt.Errorf("failed to read infra json file: %w", err)
		}
		infra = &awsinfra.CreateInfraOutput{}
		if err = json.Unmarshal(rawInfra, infra); err != nil {
			return fmt.Errorf("failed to load infra json: %w", err)
		}
	}
	if opts.BaseDomain == "" {
		if infra != nil {
			opts.BaseDomain = infra.BaseDomain
		} else {
			return fmt.Errorf("base-domain flag is required if infra-json is not provided")
		}
	}

	var iamInfo *awsinfra.CreateIAMOutput
	if len(opts.AWSPlatform.IAMJSON) > 0 {
		rawIAM, err := ioutil.ReadFile(opts.AWSPlatform.IAMJSON)
		if err != nil {
			return fmt.Errorf("failed to read iam json file: %w", err)
		}
		iamInfo = &awsinfra.CreateIAMOutput{}
		if err = json.Unmarshal(rawIAM, iamInfo); err != nil {
			return fmt.Errorf("failed to load infra json: %w", err)
		}
	} else {
		opt := awsinfra.CreateIAMOptions{
			Region:             opts.AWSPlatform.Region,
			AWSCredentialsFile: opts.AWSPlatform.AWSCredentialsFile,
			InfraID:            infra.InfraID,
			IssuerURL:          opts.AWSPlatform.IssuerURL,
			AdditionalTags:     opts.AWSPlatform.AdditionalTags,
			PrivateZoneID:      infra.PrivateZoneID,
			PublicZoneID:       infra.PublicZoneID,
			LocalZoneID:        infra.LocalZoneID,
			KMSKeyARN:          opts.AWSPlatform.EtcdKMSKeyARN,
		}
		iamInfo, err = opt.CreateIAM(ctx, client)
		if err != nil {
			return fmt.Errorf("failed to create iam: %w", err)
		}
	}

	tagMap, err := util.ParseAWSTags(opts.AWSPlatform.AdditionalTags)
	if err != nil {
		return fmt.Errorf("failed to parse additional tags: %w", err)
	}
	var tags []hyperv1.AWSResourceTag
	for k, v := range tagMap {
		tags = append(tags, hyperv1.AWSResourceTag{Key: k, Value: v})
	}

	exampleOptions.BaseDomain = infra.BaseDomain
	exampleOptions.IssuerURL = iamInfo.IssuerURL
	exampleOptions.PrivateZoneID = infra.PrivateZoneID
	exampleOptions.PublicZoneID = infra.PublicZoneID
	exampleOptions.InfraID = infraID
	exampleOptions.ExternalDNSDomain = opts.ExternalDNSDomain
	var zones []apifixtures.ExampleAWSOptionsZones
	for _, outputZone := range infra.Zones {
		zones = append(zones, apifixtures.ExampleAWSOptionsZones{
			Name:     outputZone.Name,
			SubnetID: &outputZone.SubnetID,
		})
	}
	exampleOptions.AWS = &apifixtures.ExampleAWSOptions{
		Region:             infra.Region,
		Zones:              zones,
		VPCID:              infra.VPCID,
		SecurityGroupID:    infra.SecurityGroupID,
		InstanceProfile:    iamInfo.ProfileName,
		InstanceType:       opts.AWSPlatform.InstanceType,
		Roles:              iamInfo.Roles,
		KMSProviderRoleARN: iamInfo.KMSProviderRoleARN,
		KMSKeyARN:          iamInfo.KMSKeyARN,
		RootVolumeSize:     opts.AWSPlatform.RootVolumeSize,
		RootVolumeType:     opts.AWSPlatform.RootVolumeType,
		RootVolumeIOPS:     opts.AWSPlatform.RootVolumeIOPS,
		ResourceTags:       tags,
		EndpointAccess:     opts.AWSPlatform.EndpointAccess,
		ProxyAddress:       infra.ProxyAddr,
	}
	return nil
}
