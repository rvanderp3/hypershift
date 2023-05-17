package fixtures

type ExampleVSphereOptions struct {
	VCenter           string
	Username          string
	Password          string
	Datacenter        string
	DefaultDatastore  string
	Folder            string
	Cluster           string
	ResourcePool      string
	TemplateVM        string
	NumCPUs           int32
	NumCoresPerSocket int32
	MemoryMiB         int64
	DiskSizeGB        int32
}
