package cmd

import (
	"math/rand"
	"time"
)

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits
)

// from https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang
// generates random string of length n.
func randStringBytesMaskImprSrc(n int) string {
	var src = rand.NewSource(time.Now().UnixNano())
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}


// dummyAWSConfig is to be consumed by a internally as a dummy config file
// is not expected to always be consistent with the current state of the world
// and should be deprecated in the future as libraries share resources.
var GKE_CONFIG_BYTE_ARRAY = []byte(`
---
version: v1
# These are the new definitions which are used throughout the configuration.
definitions:
  dnsConfig:
    - &defaultDns
      name: defaultDns
      kind: dns
      kubedns:
        cluster_ip: 10.35.240.10
        dns_domain: cluster.local
        namespace: kube-system
  helmConfigs:
    - &defaultHelm
      name: defaultHelm
      kind: helm
      repos:
        - name: atlas
          url: http://atlas.cnct.io
        - name: stable
          url: https://kubernetes-charts.storage.googleapis.com
      charts: []
  fabricConfigs:
    - &defaultCanalFabric
      name: defaultCanalFabric
      kind: fabric
      type: canal
      options:
        containers:
          kubePolicyController:
            version: v0.5.1
            location: calico/kube-policy-controller
          etcd:
            version: v3.0.9
            location: quay.io/coreos/etcd
          calicoCni:
            version: v1.4.2
            location: calico/cni
          calicoNode:
            version: v1.0.0-rc1
            location: quay.io/calico/node
          flannel:
            version: v0.6.1
            location: quay.io/coreos/flannel
        network:
          network: 10.128.0.0/10
          subnetLen: 22
          subnetMin: 10.128.0.0
          subnetMax: 10.191.255.255
          backend:
            type: vxlan
    - &defaultCanalFabric16
      name: defaultCanalFabric16
      kind: fabric
      type: canal
      options:
        containers:
          calicoCni:
            version: v1.10.0
            location: quay.io/calico/cni
          calicoNode:
            version: v2.4.1
            location: quay.io/calico/node
          flannel:
            version: v0.8.0
            location: quay.io/coreos/flannel
        network:
          network: 10.128.0.0/10
          subnetLen: 22
          subnetMin: 10.128.0.0
          subnetMax: 10.191.255.255
          backend:
            type: vxlan
    - &kubeVersionedFabric
      name: kubeVersionedFabric
      kind: versionedFabric
      type: canal
      kubeVersion:
        default: *defaultCanalFabric
        versions:
          v1.6: *defaultCanalFabric16
          v1.7: *defaultCanalFabric16
    - &defaultWeaveFabric
      name: defaultWeaveFabric
      kind: fabric
      type: weave
      options:
        containers:
          weave:
            version: 1.9.8
            location: weaveworks/weave-kube
          weave_npc:
            version: 1.9.8
            location: weaveworks/weave-npc
        network:
          network: 10.128.0.0/10
          nodeConnectionLimit: 30
    - &defaultWeaveFabric16
      name: defaultWeaveFabric16
      kind: fabric
      type: weave
      options:
        containers:
          weave:
            version: 1.9.8
            location: weaveworks/weave-kube
          weave_npc:
            version: 1.9.8
            location: weaveworks/weave-npc
        network:
          network: 10.128.0.0/10
          nodeConnectionLimit: 30
    - &kubeVersionedWeaveFabric
      name: kubeVersionedWeaveFabric
      kind: versionedFabric
      type: weave
      kubeVersion:
        default: *defaultWeaveFabric
        versions:
          v1.6: *defaultWeaveFabric16
          v1.7: *defaultWeaveFabric16
  kubeConfigs:
    - &defaultKube
      name: defaultKube
      kind: kubernetes
      version: v1.7.5
      hyperkubeLocation: gcr.io/google_containers/hyperkube
  containerConfigs:
    - &defaultDocker
      name: defaultDocker
      kind: container
      runtime: docker
      type: distro
  nodeConfigs:
    - &defaultGKEClusterNode
      name: defaultGKEClusterNode
      kind: node
      providerConfig:
        diskSize: 100
        machineType: n1-standard-16
        scopes:
          - https://www.googleapis.com/auth/compute
          - https://www.googleapis.com/auth/devstorage.read_only
          - https://www.googleapis.com/auth/logging.write
          - https://www.googleapis.com/auth/monitoring
    - &defaultGKEOtherNode
      name: defaultGKEOtherNode
      kind: node
      providerConfig:
        diskSize: 100
        machineType: n1-standard-1
        scopes:
          - https://www.googleapis.com/auth/compute
          - https://www.googleapis.com/auth/devstorage.read_only
          - https://www.googleapis.com/auth/logging.write
          - https://www.googleapis.com/auth/monitoring
  keyPairs:
   - &defaultGKEKeyPair
      name: defaultGKEKeyPair
      kind: keyPair
      providerConfig:
        serviceAccount: patrickrobot@k8s-work.iam.gserviceaccount.com
        serviceAccountKeyFile:  $HOME/.config/gcloud/patrickRobot.json
  kubeAuth:
   - &defaultKubeAuth
      authz: {}
      authn:
        basic:
          -
            password: "ChangeMe"
            user: "admin"
        cert:
          -
            user: "admin"
        default_basic_user: "admin"
   - &rbacKubeAuth
      authz:
        rbac:
          # super_user is required until kubernetes 1.5 is no longer supported by k2.
          # It is not used by kubernetes 1.6 or later.
          super_user: "placeholder"
      authn:
        cert:
          -
            user: "admin"
            group: "system:masters"
        default_basic_user: "admin"
  providerConfigs:
    - &defaultGKE
      name: defaultGKE
      kind: provider
      provider: gke
      type: autonomous
      project: k8s-work
      keypair: *defaultGKEKeyPair
      zone:
        primaryZone: us-east1-b
        additionalZones:
          - us-east1-c
          - us-east1-d

# This is the core of the new configuration.

deployment:
  clusters:
    - name: dummyGKEcluster
      network: 10.32.0.0/12
      dns: 10.32.0.2
      domain: cluster.local
      providerConfig: *defaultGKE
      kubeAuth: *rbacKubeAuth
      nodePools:
        - name: clusternodes
          count: 2
          kubeConfig: *defaultKube
          nodeConfig: *defaultGKEClusterNode
        - name: othernodes
          count: 2
          kubeConfig: *defaultKube
          nodeConfig: *defaultGKEOtherNode
      fabricConfig: *kubeVersionedFabric
      helmConfig: *defaultHelm
      dnsConfig: *defaultDns
      helmOverride:
  readiness:
    type: exact
    value: 0
    wait: 600
`)


var AWS_CONFIG_BYTE_ARRAY = []byte(`
---
version: v1
# These are the new definitions which are used throughout the configuration.
definitions:
  dnsConfig:
    - &defaultDns
      name: defaultDns
      kind: dns
      kubedns:
        cluster_ip: 10.32.0.2
        dns_domain: cluster.local
        namespace: kube-system

  helmConfigs:
    - &defaultHelm
      name: defaultHelm
      kind: helm
      repos:
        - name: atlas
          url: http://atlas.cnct.io
        - name: stable
          url: https://kubernetes-charts.storage.googleapis.com
      charts:
        - name: heapster
          registry: quay.io
          chart: samsung_cnct/heapster
          version: 0.1.0-0
          namespace: kube-system

  fabricConfigs:
    - &defaultCanalFabric
      name: defaultCanalFabric
      kind: fabric
      type: canal
      options:
        containers:
          kubePolicyController:
            version: v0.5.1
            location: calico/kube-policy-controller
          etcd:
            version: v3.0.9
            location: quay.io/coreos/etcd
          calicoCni:
            version: v1.4.2
            location: calico/cni
          calicoNode:
            version: v1.0.0-rc1
            location: quay.io/calico/node
          flannel:
            version: v0.6.1
            location: quay.io/coreos/flannel
        network:
          network: 10.128.0.0/10
          subnetLen: 22
          subnetMin: 10.128.0.0
          subnetMax: 10.191.255.255
          backend:
            type: vxlan
    - &defaultCanalFabric16
      name: defaultCanalFabric16
      kind: fabric
      type: canal
      options:
        containers:
          calicoCni:
            version: v1.10.0
            location: quay.io/calico/cni
          calicoNode:
            version: v2.4.1
            location: quay.io/calico/node
          flannel:
            version: v0.8.0
            location: quay.io/coreos/flannel
        network:
          network: 10.128.0.0/10
          subnetLen: 22
          subnetMin: 10.128.0.0
          subnetMax: 10.191.255.255
          backend:
            type: vxlan
    - &kubeVersionedFabric
      name: kubeVersionedFabric
      kind: versionedFabric
      type: canal
      kubeVersion:
        default: *defaultCanalFabric
        versions:
          v1.6: *defaultCanalFabric16
          v1.7: *defaultCanalFabric16
    - &defaultWeaveFabric
      name: defaultWeaveFabric
      kind: fabric
      type: weave
      options:
        containers:
          weave:
            version: 1.9.8
            location: weaveworks/weave-kube
          weave_npc:
            version: 1.9.8
            location: weaveworks/weave-npc
        network:
          network: 10.128.0.0/10
          nodeConnectionLimit: 30
    - &defaultWeaveFabric16
      name: defaultWeaveFabric16
      kind: fabric
      type: weave
      options:
        containers:
          weave:
            version: 1.9.8
            location: weaveworks/weave-kube
          weave_npc:
            version: 1.9.8
            location: weaveworks/weave-npc
        network:
          network: 10.128.0.0/10
          nodeConnectionLimit: 30
    - &kubeVersionedWeaveFabric
      name: kubeVersionedWeaveFabric
      kind: versionedFabric
      type: weave
      kubeVersion:
        default: *defaultWeaveFabric
        versions:
          v1.6: *defaultWeaveFabric16
          v1.7: *defaultWeaveFabric16
  kvStoreConfigs:
    - &defaultEtcd
      name: etcd
      kind: kvStore
      type: etcd
      uuidToken: true
      clientPorts: [2379, 4001]
      peerPorts: [2380]
      ssl: true
      version: v3.2.5
    - &defaultEtcdEvents
      name: etcdEvents
      kind: kvStore
      type: etcd
      uuidToken: true
      clientPorts: [2381]
      peerPorts: [2382]
      ssl: true
      version: v3.2.5

  apiServerConfigs:
    - &defaultApiServer
      name: defaultApiServer
      kind: apiServer
      loadBalancer: cloud
      state:
        etcd: *defaultEtcd
      events:
        etcd: *defaultEtcdEvents

  bastionConfigs:
    - &defaultBastion
      name: defaultBastion
      kind: bastion

  kubeConfigs:
    - &defaultKube
      name: defaultKube
      kind: kubernetes
      version: v1.7.6
      hyperkubeLocation: gcr.io/google_containers/hyperkube

  containerConfigs:
    - &defaultDocker
      name: defaultDocker
      kind: container
      runtime: docker
      type: distro

  osConfigs:
    - &defaultCoreOs
      name: defaultCoreOs
      kind: os
      type: coreOs
      version: 1465.7.0
      channel: stable
      # If rebootStrategy is anything other than "off" then the version of
      # CoreOS specified above will not be pinned.
      rebootStrategy: "off"
      locksmith:
        etcdConfig: *defaultEtcd
        setMax: 1
        windowStart: 04:00
        windowLength: 2h

  nodeConfigs:
    - &defaultAwsEtcdNode
      name: defaultAwsEtcdNode
      kind: node
      mounts:
        -
          device: sdf
          path: /var/lib/docker
          forceFormat: true
        -
          device: sdg
          path: /ephemeral
          forceFormat: false
      providerConfig:
        enablePublicIPs: true
        provider: aws
        # m4.16xlarge recommended for 5K scalability clusters. See goo.gl/dxQuYD
        type: m4.large
        subnet: ["zone-1", "zone-2", "zone-3"]
        tags:
          -
            key: comments
            value: "Cluster etcd"
        storage:
          -
            type: ebs_block_device
            opts:
              device_name: sdf
              volume_type: gp2
              volume_size: 100
              delete_on_termination: true
              snapshot_id:
              encrypted: false
          -
            type: ebs_block_device
            opts:
              device_name: sdg
              volume_type: gp2
              volume_size: 10
              delete_on_termination: true
              snapshot_id:
              encrypted: false
    - &defaultAwsEtcdEventsNode
      name: defaultAwsEtcdEventsNode
      kind: node
      mounts:
        -
          device: sdf
          path: /var/lib/docker
          forceFormat: true
        -
          device: sdg
          path: /ephemeral
          forceFormat: false
      providerConfig:
        enablePublicIPs: true
        provider: aws
        # m4.16xlarge recommended for 5K scalability clusters. See goo.gl/dxQuYD
        type: m4.large
        subnet: ["zone-1", "zone-2", "zone-3"]
        tags:
          -
            key: comments
            value: "Cluster events etcd"
        storage:
          -
            type: ebs_block_device
            opts:
              device_name: sdf
              volume_type: gp2
              volume_size: 100
              delete_on_termination: true
              snapshot_id:
              encrypted: false
          -
            type: ebs_block_device
            opts:
              device_name: sdg
              volume_type: gp2
              volume_size: 10
              delete_on_termination: true
              snapshot_id:
              encrypted: false
    - &defaultAwsMasterNode
      name: defaultAwsMasterNode
      kind: node
      mounts:
        -
          device: sdf
          path: /var/lib/docker
          forceFormat: true
      providerConfig:
        enablePublicIPs: true
        provider: aws
        type: m4.large
        subnet: ["zone-1", "zone-2", "zone-3"]
        tags:
          -
            key: comments
            value: "Master instances"
        storage:
          -
            type: ebs_block_device
            opts:
              device_name: sdf
              volume_type: gp2
              volume_size: 100
              delete_on_termination: true
              snapshot_id:
              encrypted: false
    - &defaultAwsClusterNode
      name: defaultAwsClusterNode
      kind: node
      mounts:
        -
          device: sdf
          path: /var/lib/docker
          forceFormat: true
      providerConfig:
        enablePublicIPs: true
        provider: aws
        type: c4.large
        subnet: ["zone-1", "zone-2", "zone-3"]
        tags:
          -
            key: comments
            value: "Cluster plain nodes"
        storage:
          -
            type: ebs_block_device
            opts:
              device_name: sdf
              volume_type: gp2
              volume_size: 100
              delete_on_termination: true
              snapshot_id:
              encrypted: false
    - &defaultAwsSpecialNode
      name: defaultAwsSpecialNode
      kind: node
      mounts:
        -
          device: sdf
          path: /var/lib/docker
          forceFormat: true
      keypair: krakenKey
      providerConfig:
        provider: aws
        enablePublicIPs: true
        type: m4.large
        subnet: ["zone-1", "zone-2", "zone-3"]
        tags:
          -
            key: comments
            value: "Cluster special nodes"
        storage:
          -
            type: ebs_block_device
            opts:
              device_name: sdf
              volume_type: gp2
              volume_size: 100
              delete_on_termination: true
              snapshot_id:
              encrypted: false

  providerConfigs:
    - &defaultAws
      name: defaultAws
      kind: provider
      provider: aws
      type: cloudinit
      resourcePrefix:
      vpc: 10.0.0.0/16
      region: us-east-1
      subnet:
        -
          name: zone-1
          az: us-east-1a
          cidr: 10.0.0.0/18
          enablePublicIPs: true
        -
          name: zone-2
          az: us-east-1b
          cidr: 10.0.64.0/18
          enablePublicIPs: true
        -
          name: zone-3
          az: us-east-1c
          cidr: 10.0.128.0/17
          enablePublicIPs: true
      egressAcl:
        -
          protocol: "-1"
          rule_no: 100
          action: "allow"
          cidr_block: "0.0.0.0/0"
          from_port: 0
          to_port: 0
      ingressAcl:
        -
          protocol: "-1"
          rule_no: 100
          action: "allow"
          cidr_block: "0.0.0.0/0"
          from_port: 0
          to_port: 0
      authentication:
        accessKey:
        accessSecret:
        credentialsFile: "$HOME/.aws/credentials"
        credentialsProfile:
      ingressSecurity:
        -
          from_port: 22
          to_port: 22
          protocol: "TCP"
          cidr_blocks: ["0.0.0.0/0"]
        -
          from_port: 443
          to_port: 443
          protocol: "TCP"
          cidr_blocks: ["0.0.0.0/0"]
      egressSecurity:
        -
          from_port: 0
          to_port: 0
          protocol: "-1"
          cidr_blocks: ["0.0.0.0/0"]

  keyPairs:
   - &defaultKeyPair
      name: defaultKeyPair
      kind: keyPair
      publickeyFile: "$HOME/.ssh/id_rsa.pub"
      privatekeyFile: "$HOME/.ssh/id_rsa"

  kubeAuth:
   - &defaultKubeAuth
      authz: {}
      authn:
        basic:
          -
            password: "ChangeMe"
            user: "admin"
        cert:
          -
            user: "admin"
        default_basic_user: "admin"
   - &rbacKubeAuth
      authz:
        rbac:
          # super_user is required until kubernetes 1.5 is no longer supported by k2.
          # It is not used by kubernetes 1.6 or later.
          super_user: "placeholder"
      authn:
        cert:
          -
            user: "admin"
            group: "system:masters"
        default_basic_user: "admin"

# This is the core of the new configuration.

deployment:
  clusters:
    - name: dummyAWScluster
      network: 10.32.0.0/12
      dns: 10.32.0.2
      domain: cluster.local
      providerConfig: *defaultAws
      nodePools:
        - name: etcd
          # 5 nodes recommended for production/high availability clusters. This
          # configuration is required to tolerate a single node failure during
          # node upgrades.
          count: 5
          etcdConfig: *defaultEtcd
          containerConfig: *defaultDocker
          osConfig: *defaultCoreOs
          nodeConfig: *defaultAwsEtcdNode
          keyPair: *defaultKeyPair
        - name: etcdEvents
          # 5 nodes recommended for production/high availability clusters. This
          # configuration is required to tolerate a single node failure during
          # node upgrades.
          count: 5
          etcdConfig: *defaultEtcdEvents
          containerConfig: *defaultDocker
          osConfig: *defaultCoreOs
          nodeConfig: *defaultAwsEtcdEventsNode
          keyPair: *defaultKeyPair
        - name: master
          count: 3
          apiServerConfig: *defaultApiServer
          kubeConfig: *defaultKube
          containerConfig: *defaultDocker
          osConfig: *defaultCoreOs
          nodeConfig: *defaultAwsMasterNode
          keyPair: *defaultKeyPair
        - name: clusterNodes
          count: 10
          kubeConfig: *defaultKube
          containerConfig: *defaultDocker
          osConfig: *defaultCoreOs
          nodeConfig: *defaultAwsClusterNode
          keyPair: *defaultKeyPair
        - name: specialNodes
          count: 2
          # m4.large
          kubeConfig: *defaultKube
          containerConfig: *defaultDocker
          osConfig: *defaultCoreOs
          nodeConfig: *defaultAwsSpecialNode
          keyPair: *defaultKeyPair
      fabricConfig: *kubeVersionedFabric
      kubeAuth: *rbacKubeAuth
      helmConfig: *defaultHelm
      dnsConfig: *defaultDns
      # customApiDns:
      helmOverride:
  readiness:
    type: exact
    value: 0
    wait: 600
`)
