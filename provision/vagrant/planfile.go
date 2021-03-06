package vagrant

import (
	"bufio"
	"html/template"
	"os"
)

type PlanOpts struct {
	InfrastructureOpts
	DisablePackageInstallation   bool
	AutoConfiguredDockerRegistry bool
	DockerRegistryHost           string
	DockerRegistryPort           uint16
	DockerRegistryCAPath         string
	PodCIDR                      string
	ServiceCIDR                  string
}

type Plan struct {
	Opts           *PlanOpts
	Infrastructure *Infrastructure
}

func (p *Plan) Write(file *os.File) error {
	template, err := template.New("planVagrantOverlay").Parse(planVagrantOverlay)
	if err != nil {
		return err
	}

	w := bufio.NewWriter(file)

	if err = template.Execute(w, &p); err != nil {
		return err
	}

	w.Flush()

	return nil
}

func (p *Plan) Etcd() []NodeDetails {
	return p.Infrastructure.nodesByType(Etcd)
}

func (p *Plan) Master() []NodeDetails {
	return p.Infrastructure.nodesByType(Master)
}

func (p *Plan) Worker() []NodeDetails {
	return p.Infrastructure.nodesByType(Worker)
}

func (p *Plan) Ingress() []NodeDetails {
	return p.Infrastructure.nodesByType(Worker)[0:1]
}

func (p *Plan) Storage() []NodeDetails {
	if p.Opts.Storage {
		return p.Infrastructure.nodesByType(Worker)
	}
	return []NodeDetails{}
}

const planVagrantOverlay = `cluster:
  name: kubernetes

  # Set to true if the nodes have the required packages installed.
  disable_package_installation: {{.Opts.DisablePackageInstallation}}

  # Set to true if you are performing a disconnected installation.
  disconnected_installation: false

  # Networking configuration of your cluster.
  networking:

    # Kubernetes will assign pods IPs in this range. Do not use a range that is
    # already in use on your local network!
    pod_cidr_block: {{.Opts.PodCIDR}}

    # Kubernetes will assign services IPs in this range. Do not use a range
    # that is already in use by your local network or pod network!
    service_cidr_block: {{.Opts.ServiceCIDR}}

    # Set to true if your nodes cannot resolve each others' names using DNS.
    update_hosts_files: true

    # Set the proxy server to use for HTTP connections.
    http_proxy: ""

    # Set the proxy server to use for HTTPs connections.
    https_proxy: ""

    # List of host names and/or IPs that shouldn't go through any proxy.
    # All nodes' 'host' and 'IPs' are always set.
    no_proxy: ""

  # Generated certs configuration.
  certificates:

    # Self-signed certificate expiration period in hours; default is 2 years.
    expiry: 17520h

    # CA certificate expiration period in hours; default is 2 years.
    ca_expiry: 17520h

  # SSH configuration for cluster nodes.
  ssh:

    # This user must be able to sudo without password.
    user: vagrant

    # Absolute path to the ssh private key we should use to manage nodes.
    ssh_key: {{.Infrastructure.PrivateSSHKeyPath}}
    ssh_port: 22

  # Override configuration of Kubernetes components.
  kube_apiserver:
    option_overrides: {}

  kube_controller_manager:
    option_overrides: {}

  kube_scheduler:
    option_overrides: {}

  kube_proxy:
    option_overrides: {}

  kubelet:
    option_overrides: {}

  # Kubernetes cloud provider integration
  cloud_provider:

    # Options: 'aws','azure','cloudstack','fake','gce','mesos','openstack',
    # 'ovirt','photon','rackspace','vsphere'.
    # Leave empty for bare metal setups or other unsupported providers.
    provider: ""

    # Path to the config file, leave empty if provider does not require it.
    config: ""

# Docker daemon configuration of all cluster nodes
docker:
  logs:
    driver: json-file
    opts:
      max-file: "1"
      max-size: 50m

  storage:

    # Configure devicemapper in direct-lvm mode (RHEL/CentOS only).
    direct_lvm:
      enabled: false

      # Path to the block device that will be used for direct-lvm mode. This
      # device will be wiped and used exclusively by docker.
      block_device: ""

      # Set to true if you want to enable deferred deletion when using
      # direct-lvm mode.
      enable_deferred_deletion: false

# If you want to use an internal registry for the installation or upgrade, you
# must provide its information here. You must seed this registry before the
# installation or upgrade of your cluster. This registry must be accessible from
# all nodes on the cluster.
docker_registry:

  # IP or hostname and port for your registry.
  server: ""

  # Absolute path to the certificate authority that should be trusted when
  # connecting to your registry.
  CA: {{.Opts.DockerRegistryCAPath}}

  # Leave blank for unauthenticated access.
  username: ""

  # Leave blank for unauthenticated access.
  password: ""

# Add-ons are additional components that KET installs on the cluster.
add_ons:
  cni:
    disable: false

    # Selecting 'custom' will result in a CNI ready cluster, however it is up to
    # you to configure a plugin after the install.
    # Options: 'calico','weave','contiv','custom'.
    provider: calico
    options:
      calico:

        # Options: 'overlay','routed'.
        mode: overlay

        # Options: 'warning','info','debug'.
        log_level: info

        # MTU for the workload interface, configures the CNI config.
        workload_mtu: 1500

        # MTU for the tunnel device used if IPIP is enabled.
        felix_input_mtu: 1440

  dns:
    disable: false

  heapster:
    disable: false
    options:
      heapster:
        replicas: 2

        # Specify kubernetes ServiceType. Defaults to 'ClusterIP'.
        # Options: 'ClusterIP','NodePort','LoadBalancer','ExternalName'.
        service_type: ClusterIP

        # Specify the sink to store heapster data. Defaults to an influxdb pod
        # running on the cluster.
        sink: influxdb:http://heapster-influxdb.kube-system.svc:8086

      influxdb:

        # Provide the name of the persistent volume claim that you will create
        # after installation. If not specified, the data will be stored in
        # ephemeral storage.
        pvc_name: ""

  dashboard:
    disable: false

  package_manager:
    disable: false

    # Options: 'helm'
    provider: helm

  # The rescheduler ensures that critical add-ons remain running on the cluster.
  rescheduler:
    disable: false

# Etcd nodes are the ones that run the etcd distributed key-value database.
etcd:
  expected_count: {{len .Etcd}}

  # Provide the hostname and IP of each node. If the node has an IP for internal
  # traffic, provide it in the internalip field. Otherwise, that field can be
  # left blank.
  nodes:{{range .Etcd}}
  - host: {{.Name}}
    ip: {{.IP.String}}
    labels: {}{{end}}

# Master nodes are the ones that run the Kubernetes control plane components.
master:
  expected_count: {{len .Master}}

  # If you have set up load balancing for master nodes, enter the FQDN name here.
  # Otherwise, use the IP address of a single master node.
  load_balanced_fqdn: {{with index .Master 0 }}{{.IP.String}}{{end}}

  # If you have set up load balancing for master nodes, enter the short name here.
  # Otherwise, use the IP address of a single master node.
  load_balanced_short_name: {{with index .Master 0}}{{.IP.String}}{{end}}
  nodes:{{range .Master}}
  - host: {{.Name}}
    ip: {{.IP.String}}
    labels: {}{{end}}

# Worker nodes are the ones that will run your workloads on the cluster.
worker:
  expected_count: {{len .Worker}}
  nodes:{{range .Worker}}
  - host: {{.Name}}
    ip: {{.IP.String}}
    labels: {}{{end}}

# Ingress nodes will run the ingress controllers.
ingress:
  expected_count: {{len .Ingress}}
  nodes:{{range .Ingress}}
  - host: {{.Name}}
    ip: {{.IP.String}}
    labels: {}{{end}}

# Storage nodes will be used to create a distributed storage cluster that can
# be consumed by your workloads.
storage:
  expected_count: {{len .Storage}}
  nodes:{{range .Storage}}
  - host: {{.Name}}
    ip: {{.IP.String}}
    labels: {}{{end}}
`
