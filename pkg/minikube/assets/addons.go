/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package assets

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"k8s.io/minikube/deploy/addons"
	"k8s.io/minikube/pkg/minikube/config"
	"k8s.io/minikube/pkg/minikube/constants"
	"k8s.io/minikube/pkg/minikube/out"
	"k8s.io/minikube/pkg/minikube/vmpath"
	"k8s.io/minikube/pkg/util"
	"k8s.io/minikube/pkg/version"
)

// Addon is a named list of assets, that can be enabled
type Addon struct {
	Assets     []*BinAsset
	enabled    bool
	addonName  string
	Maintainer string
	Images     map[string]string

	// Registries currently only shows the default registry of images
	Registries map[string]string
}

// NetworkInfo contains control plane node IP address used for add on template
type NetworkInfo struct {
	ControlPlaneNodeIP   string
	ControlPlaneNodePort int
}

// NewAddon creates a new Addon
func NewAddon(assets []*BinAsset, enabled bool, addonName string, maintainer string, images map[string]string, registries map[string]string) *Addon {
	a := &Addon{
		Assets:     assets,
		enabled:    enabled,
		addonName:  addonName,
		Maintainer: maintainer,
		Images:     images,
		Registries: registries,
	}
	return a
}

// Name get the addon name
func (a *Addon) Name() string {
	return a.addonName
}

// IsEnabled checks if an Addon is enabled for the given profile
func (a *Addon) IsEnabled(cc *config.ClusterConfig) bool {
	status, ok := cc.Addons[a.Name()]
	if ok {
		return status
	}

	// Return the default unconfigured state of the addon
	return a.enabled
}

// Addons is the list of addons
// TODO: Make dynamically loadable: move this data to a .yaml file within each addon directory
var Addons = map[string]*Addon{
	"auto-pause": NewAddon([]*BinAsset{
		MustBinAsset(
			addons.AutoPauseAssets,
			"auto-pause/auto-pause.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"auto-pause.yaml",
			"0640"),
		MustBinAsset(
			addons.AutoPauseAssets,
			"auto-pause/auto-pause-hook.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"auto-pause-hook.yaml",
			"0640"),
		MustBinAsset(
			addons.AutoPauseAssets,
			"auto-pause/haproxy.cfg.tmpl",
			vmpath.GuestPersistentDir,
			"haproxy.cfg",
			"0640"),
		MustBinAsset(
			addons.AutoPauseAssets,
			"auto-pause/unpause.lua",
			vmpath.GuestPersistentDir,
			"unpause.lua",
			"0640"),
		MustBinAsset(
			addons.AutoPauseAssets,
			"auto-pause/auto-pause.service.tmpl",
			"/etc/systemd/system/",
			"auto-pause.service",
			"0640"),

		// GuestPersistentDir
	}, false, "auto-pause", "google", map[string]string{
		"AutoPauseHook": "k8s-minikube/auto-pause-hook:v0.0.2@sha256:c76be418df5ca9c66d0d11c2c68461acbf4072c1cdfc17e64729c5ef4d5a4128",
	}, map[string]string{
		"AutoPauseHook": "gcr.io",
	}),
	"dashboard": NewAddon([]*BinAsset{
		// We want to create the kubernetes-dashboard ns first so that every subsequent object can be created
		MustBinAsset(addons.DashboardAssets, "dashboard/dashboard-ns.yaml", vmpath.GuestAddonsDir, "dashboard-ns.yaml", "0640"),
		MustBinAsset(addons.DashboardAssets, "dashboard/dashboard-clusterrole.yaml", vmpath.GuestAddonsDir, "dashboard-clusterrole.yaml", "0640"),
		MustBinAsset(addons.DashboardAssets, "dashboard/dashboard-clusterrolebinding.yaml", vmpath.GuestAddonsDir, "dashboard-clusterrolebinding.yaml", "0640"),
		MustBinAsset(addons.DashboardAssets, "dashboard/dashboard-configmap.yaml", vmpath.GuestAddonsDir, "dashboard-configmap.yaml", "0640"),
		MustBinAsset(addons.DashboardAssets, "dashboard/dashboard-dp.yaml.tmpl", vmpath.GuestAddonsDir, "dashboard-dp.yaml", "0640"),
		MustBinAsset(addons.DashboardAssets, "dashboard/dashboard-role.yaml", vmpath.GuestAddonsDir, "dashboard-role.yaml", "0640"),
		MustBinAsset(addons.DashboardAssets, "dashboard/dashboard-rolebinding.yaml", vmpath.GuestAddonsDir, "dashboard-rolebinding.yaml", "0640"),
		MustBinAsset(addons.DashboardAssets, "dashboard/dashboard-sa.yaml", vmpath.GuestAddonsDir, "dashboard-sa.yaml", "0640"),
		MustBinAsset(addons.DashboardAssets, "dashboard/dashboard-secret.yaml", vmpath.GuestAddonsDir, "dashboard-secret.yaml", "0640"),
		MustBinAsset(addons.DashboardAssets, "dashboard/dashboard-svc.yaml", vmpath.GuestAddonsDir, "dashboard-svc.yaml", "0640"),
	}, false, "dashboard", "kubernetes", map[string]string{
		"Dashboard":      "kubernetesui/dashboard:v2.5.1@sha256:cc746e7a0b1eec0db01cbabbb6386b23d7af97e79fa9e36bb883a95b7eb96fe2",
		"MetricsScraper": "kubernetesui/metrics-scraper:v1.0.7@sha256:36d5b3f60e1a144cc5ada820910535074bdf5cf73fb70d1ff1681537eef4e172",
	}, nil),
	"default-storageclass": NewAddon([]*BinAsset{
		MustBinAsset(addons.DefaultStorageClassAssets,
			"storageclass/storageclass.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"storageclass.yaml",
			"0640"),
	}, true, "default-storageclass", "kubernetes", nil, nil),
	"pod-security-policy": NewAddon([]*BinAsset{
		MustBinAsset(addons.PodSecurityPolicyAssets,
			"pod-security-policy/pod-security-policy.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"pod-security-policy.yaml",
			"0640"),
	}, false, "pod-security-policy", "", nil, nil),
	"storage-provisioner": NewAddon([]*BinAsset{
		MustBinAsset(addons.StorageProvisionerAssets,
			"storage-provisioner/storage-provisioner.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"storage-provisioner.yaml",
			"0640"),
	}, true, "storage-provisioner", "google", map[string]string{
		"StorageProvisioner": fmt.Sprintf("k8s-minikube/storage-provisioner:%s", version.GetStorageProvisionerVersion()),
	}, map[string]string{
		"StorageProvisioner": "gcr.io",
	}),
	"storage-provisioner-gluster": NewAddon([]*BinAsset{
		MustBinAsset(addons.StorageProvisionerGlusterAssets,
			"storage-provisioner-gluster/storage-gluster-ns.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"storage-gluster-ns.yaml",
			"0640"),
		MustBinAsset(addons.StorageProvisionerGlusterAssets,
			"storage-provisioner-gluster/glusterfs-daemonset.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"glusterfs-daemonset.yaml",
			"0640"),
		MustBinAsset(addons.StorageProvisionerGlusterAssets,
			"storage-provisioner-gluster/heketi-deployment.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"heketi-deployment.yaml",
			"0640"),
		MustBinAsset(addons.StorageProvisionerGlusterAssets,
			"storage-provisioner-gluster/storage-provisioner-glusterfile.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"storage-provisioner-glusterfile.yaml",
			"0640"),
	}, false, "storage-provisioner-gluster", "", map[string]string{
		"Heketi":                 "heketi/heketi:10@sha256:76d5a6a3b7cf083d1e99efa1c15abedbc5c8b73bef3ade299ce9a4c16c9660f8",
		"GlusterfileProvisioner": "gluster/glusterfile-provisioner:latest@sha256:9961a35cb3f06701958e202324141c30024b195579e5eb1704599659ddea5223",
		"GlusterfsServer":        "nixpanic/glusterfs-server:pr_fake-disk@sha256:3c58ae9d4e2007758954879d3f4095533831eb757c64ca6a0e32d1fc53fb6034",
	}, map[string]string{
		"GlusterfsServer": "quay.io",
	}),
	"efk": NewAddon([]*BinAsset{
		MustBinAsset(addons.EfkAssets,
			"efk/elasticsearch-rc.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"elasticsearch-rc.yaml",
			"0640"),
		MustBinAsset(addons.EfkAssets,
			"efk/elasticsearch-svc.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"elasticsearch-svc.yaml",
			"0640"),
		MustBinAsset(addons.EfkAssets,
			"efk/fluentd-es-rc.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"fluentd-es-rc.yaml",
			"0640"),
		MustBinAsset(addons.EfkAssets,
			"efk/fluentd-es-configmap.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"fluentd-es-configmap.yaml",
			"0640"),
		MustBinAsset(addons.EfkAssets,
			"efk/kibana-rc.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"kibana-rc.yaml",
			"0640"),
		MustBinAsset(addons.EfkAssets,
			"efk/kibana-svc.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"kibana-svc.yaml",
			"0640"),
	}, false, "efk", "third-party (elastic)", map[string]string{
		"Elasticsearch":        "elasticsearch:v5.6.2@sha256:7e95b32a7a2aad0c0db5c881e4a1ce8b7e53236144ae9d9cfb5fbe5608af4ab2",
		"FluentdElasticsearch": "fluentd-elasticsearch:v2.0.2@sha256:d0480bbf2d0de2344036fa3f7034cf7b4b98025a89c71d7f1f1845ac0e7d5a97",
		"Alpine":               "alpine:3.6@sha256:66790a2b79e1ea3e1dabac43990c54aca5d1ddf268d9a5a0285e4167c8b24475",
		"Kibana":               "kibana/kibana:5.6.2@sha256:cd948a9bda4622f1437afc4a3e78be6c8c25fc62f40aa0376f3d690f2436568f",
	}, map[string]string{
		"Elasticsearch":        "k8s.gcr.io",
		"FluentdElasticsearch": "k8s.gcr.io",
		"Kibana":               "docker.elastic.co",
	}),
	"ingress": NewAddon([]*BinAsset{
		MustBinAsset(addons.IngressAssets,
			"ingress/ingress-deploy.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"ingress-deploy.yaml",
			"0640"),
	}, false, "ingress", "", map[string]string{
		// https://github.com/kubernetes/ingress-nginx/blob/6d9a39eda7b180f27b34726d7a7a96d73808ce75/deploy/static/provider/kind/deploy.yaml#L417
		"IngressController": "ingress-nginx/controller:v1.2.0@sha256:d8196e3bc1e72547c5dec66d6556c0ff92a23f6d0919b206be170bc90d5f9185",
		// https://github.com/kubernetes/ingress-nginx/blob/fc38b9f2aa2d68ee00c417cf97e727b77a00c175/deploy/static/provider/kind/deploy.yaml#L621
		"KubeWebhookCertgenCreate": "k8s.gcr.io/ingress-nginx/kube-webhook-certgen:v1.1.1@sha256:64d8c73dca984af206adf9d6d7e46aa550362b1d7a01f3a0a91b20cc67868660",
		// https://github.com/kubernetes/ingress-nginx/blob/fc38b9f2aa2d68ee00c417cf97e727b77a00c175/deploy/static/provider/kind/deploy.yaml#L673
		"KubeWebhookCertgenPatch": "k8s.gcr.io/ingress-nginx/kube-webhook-certgen:v1.1.1@sha256:64d8c73dca984af206adf9d6d7e46aa550362b1d7a01f3a0a91b20cc67868660",
	}, map[string]string{
		"IngressController": "k8s.gcr.io",
	}),
	"istio-provisioner": NewAddon([]*BinAsset{
		MustBinAsset(addons.IstioProvisionerAssets,
			"istio-provisioner/istio-operator.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"istio-operator.yaml",
			"0640"),
	}, false, "istio-provisioner", "third-party (istio)", map[string]string{
		"IstioOperator": "istio/operator:1.12.2@sha256:42c7609872882cb88728a1592561b4046dac6d05b6002cbdc815b84c86a24f08",
	}, nil),
	"istio": NewAddon([]*BinAsset{
		MustBinAsset(addons.IstioAssets,
			"istio/istio-default-profile.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"istio-default-profile.yaml",
			"0640"),
	}, false, "istio", "third-party (istio)", nil, nil),
	"kong": NewAddon([]*BinAsset{
		MustBinAsset(addons.KongAssets,
			"kong/kong-ingress-controller.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"kong-ingress-controller.yaml",
			"0640"),
	}, false, "kong", "third-party (Kong HQ)", map[string]string{
		"Kong":        "kong:2.7@sha256:4d3e93207305ace881fe9e95ac27717b6fbdd9e0ec1873c34e94908a4f4c9335",
		"KongIngress": "kong/kubernetes-ingress-controller:2.1.1@sha256:60e4102ab2da7f61e9c478747f0762d06a6166b5f300526b237ed7354c3cb4c8",
	}, nil),
	"kubevirt": NewAddon([]*BinAsset{
		MustBinAsset(addons.KubevirtAssets,
			"kubevirt/pod.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"pod.yaml",
			"0640"),
	}, false, "kubevirt", "third-party (kubevirt)", map[string]string{
		"Kubectl": "bitnami/kubectl:1.17@sha256:de642e973d3d0ef60e4d0a1f92286a9fdae245535c5990d4762bbe86fcf95887",
	}, nil),
	"metrics-server": NewAddon([]*BinAsset{
		MustBinAsset(addons.MetricsServerAssets,
			"metrics-server/metrics-apiservice.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"metrics-apiservice.yaml",
			"0640"),
		MustBinAsset(addons.MetricsServerAssets,
			"metrics-server/metrics-server-deployment.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"metrics-server-deployment.yaml",
			"0640"),
		MustBinAsset(addons.MetricsServerAssets,
			"metrics-server/metrics-server-rbac.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"metrics-server-rbac.yaml",
			"0640"),
		MustBinAsset(addons.MetricsServerAssets,
			"metrics-server/metrics-server-service.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"metrics-server-service.yaml",
			"0640"),
	}, false, "metrics-server", "kubernetes", map[string]string{
		"MetricsServer": "metrics-server/metrics-server:v0.6.1@sha256:5ddc6458eb95f5c70bd13fdab90cbd7d6ad1066e5b528ad1dcb28b76c5fb2f00",
	}, map[string]string{
		"MetricsServer": "k8s.gcr.io",
	}),
	"olm": NewAddon([]*BinAsset{
		MustBinAsset(addons.OlmAssets,
			"olm/crds.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"crds.yaml",
			"0640"),
		MustBinAsset(addons.OlmAssets,
			"olm/olm.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"olm.yaml",
			"0640"),
	}, false, "olm", "third-party (operator framework)", map[string]string{
		"OLM": "operator-framework/olm@sha256:e74b2ac57963c7f3ba19122a8c31c9f2a0deb3c0c5cac9e5323ccffd0ca198ed",
		// operator-framework/community-operators was deprecated: https://github.com/operator-framework/community-operators#repository-is-obsolete; switching to OperatorHub.io instead
		"UpstreamCommunityOperators": "operatorhubio/catalog@sha256:e08a1cd21fe72dd1be92be738b4bf1515298206dac5479c17a4b3ed119e30bd4",
	}, map[string]string{
		"OLM":                        "quay.io",
		"UpstreamCommunityOperators": "quay.io",
	}),
	"registry": NewAddon([]*BinAsset{
		MustBinAsset(addons.RegistryAssets,
			"registry/registry-rc.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"registry-rc.yaml",
			"0640"),
		MustBinAsset(addons.RegistryAssets,
			"registry/registry-svc.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"registry-svc.yaml",
			"0640"),
		MustBinAsset(addons.RegistryAssets,
			"registry/registry-proxy.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"registry-proxy.yaml",
			"0640"),
	}, false, "registry", "google", map[string]string{
		"Registry":          "registry:2.7.1@sha256:d5459fcb27aecc752520df4b492b08358a1912fcdfa454f7d2101d4b09991daa",
		"KubeRegistryProxy": "google_containers/kube-registry-proxy:0.4@sha256:1040f25a5273de0d72c54865a8efd47e3292de9fb8e5353e3fa76736b854f2da",
	}, map[string]string{
		"KubeRegistryProxy": "gcr.io",
	}),
	"registry-creds": NewAddon([]*BinAsset{
		MustBinAsset(addons.RegistryCredsAssets,
			"registry-creds/registry-creds-rc.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"registry-creds-rc.yaml",
			"0640"),
	}, false, "registry-creds", "third-party (upmc enterprises)", map[string]string{
		"RegistryCreds": "upmcenterprises/registry-creds:1.10@sha256:93a633d4f2b76a1c66bf19c664dbddc56093a543de6d54320f19f585ccd7d605",
	}, nil),
	"registry-aliases": NewAddon([]*BinAsset{
		MustBinAsset(addons.RegistryAliasesAssets,
			"registry-aliases/registry-aliases-sa.tmpl",
			vmpath.GuestAddonsDir,
			"registry-aliases-sa.yaml",
			"0640"),
		MustBinAsset(addons.RegistryAliasesAssets,
			"registry-aliases/registry-aliases-sa-crb.tmpl",
			vmpath.GuestAddonsDir,
			"registry-aliases-sa-crb.yaml",
			"0640"),
		MustBinAsset(addons.RegistryAliasesAssets,
			"registry-aliases/registry-aliases-config.tmpl",
			vmpath.GuestAddonsDir,
			"registry-aliases-config.yaml",
			"0640"),
		MustBinAsset(addons.RegistryAliasesAssets,
			"registry-aliases/node-etc-hosts-update.tmpl",
			vmpath.GuestAddonsDir,
			"node-etc-hosts-update.yaml",
			"0640"),
		MustBinAsset(addons.RegistryAliasesAssets,
			"registry-aliases/patch-coredns-job.tmpl",
			vmpath.GuestAddonsDir,
			"patch-coredns-job.yaml",
			"0640"),
	}, false, "registry-aliases", "", map[string]string{
		"CoreDNSPatcher": "rhdevelopers/core-dns-patcher@sha256:9220ff32f690c3d889a52afb59ca6fcbbdbd99e5370550cc6fd249adea8ed0a9",
		"Alpine":         "alpine:3.11@sha256:0bd0e9e03a022c3b0226667621da84fc9bf562a9056130424b5bfbd8bcb0397f",
		"Pause":          "google_containers/pause:3.1@sha256:f78411e19d84a252e53bff71a4407a5686c46983a2c2eeed83929b888179acea",
	}, map[string]string{
		"CoreDNSPatcher": "quay.io",
		"Pause":          "gcr.io",
	}),
	"freshpod": NewAddon([]*BinAsset{
		MustBinAsset(addons.FreshpodAssets,
			"freshpod/freshpod-rc.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"freshpod-rc.yaml",
			"0640"),
	}, false, "freshpod", "google", map[string]string{
		"FreshPod": "google-samples/freshpod:v0.0.1@sha256:b9efde5b509da3fd2959519c4147b653d0c5cefe8a00314e2888e35ecbcb46f9",
	}, map[string]string{
		"FreshPod": "gcr.io",
	}),
	"nvidia-driver-installer": NewAddon([]*BinAsset{
		MustBinAsset(addons.NvidiaDriverInstallerAssets,
			"gpu/nvidia-driver-installer.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"nvidia-driver-installer.yaml",
			"0640"),
	}, false, "nvidia-driver-installer", "google", map[string]string{
		"NvidiaDriverInstaller": "minikube-nvidia-driver-installer:e2d9b43228decf5d6f7dce3f0a85d390f138fa01",
		"Pause":                 "pause:2.0@sha256:9ce5316f9752b8347484ab0f6778573af15524124d52b93230b9a0dcc987e73e",
	}, map[string]string{
		"NvidiaDriverInstaller": "k8s.gcr.io",
		"Pause":                 "k8s.gcr.io",
	}),
	"nvidia-gpu-device-plugin": NewAddon([]*BinAsset{
		MustBinAsset(addons.NvidiaGpuDevicePluginAssets,
			"gpu/nvidia-gpu-device-plugin.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"nvidia-gpu-device-plugin.yaml",
			"0640"),
	}, false, "nvidia-gpu-device-plugin", "third-party (nvidia)", map[string]string{
		"NvidiaDevicePlugin": "nvidia-gpu-device-plugin@sha256:4b036e8844920336fa48f36edeb7d4398f426d6a934ba022848deed2edbf09aa",
	}, map[string]string{
		"NvidiaDevicePlugin": "k8s.gcr.io",
	}),
	"logviewer": NewAddon([]*BinAsset{
		MustBinAsset(addons.LogviewerAssets,
			"logviewer/logviewer-dp-and-svc.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"logviewer-dp-and-svc.yaml",
			"0640"),
		MustBinAsset(addons.LogviewerAssets,
			"logviewer/logviewer-rbac.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"logviewer-rbac.yaml",
			"0640"),
	}, false, "logviewer", "", map[string]string{
		"LogViewer": "ivans3/minikube-log-viewer:latest@sha256:75854f45305cc47d17b04c6c588fa60777391761f951e3a34161ddf1f1b06405",
	}, nil),
	"gvisor": NewAddon([]*BinAsset{
		MustBinAsset(addons.GvisorAssets,
			"gvisor/gvisor-pod.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"gvisor-pod.yaml",
			"0640"),
		MustBinAsset(addons.GvisorAssets,
			"gvisor/gvisor-runtimeclass.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"gvisor-runtimeclass.yaml",
			"0640"),
		MustBinAsset(addons.GvisorAssets,
			"gvisor/gvisor-config.toml",
			vmpath.GuestGvisorDir,
			constants.GvisorConfigTomlTargetName,
			"0640"),
	}, false, "gvisor", "google", map[string]string{
		"GvisorAddon": "k8s-minikube/gvisor-addon:3@sha256:23eb17d48a66fc2b09c31454fb54ecae520c3e9c9197ef17fcb398b4f31d505a",
	}, map[string]string{
		"GvisorAddon": "gcr.io",
	}),
        "xspot": NewAddon([]*BinAsset{
                MustBinAsset(addons.XSpotAssets,
                        "xspot/xspot-pod.yaml.tmpl",
                        vmpath.GuestAddonsDir,
                        "xspot-pod.yaml",
                        "0640"),
                MustBinAsset(addons.XSpotAssets,
                        "xspot/xspot-runtimeclass.yaml.tmpl",
                        vmpath.GuestAddonsDir,
                        "xspot-runtimeclass.yaml",
                        "0640"),
                MustBinAsset(addons.XSpotAssets,
                        "xspot/xspot-config.toml",
                        vmpath.GuestXSpotDir,
                        constants.XSpotConfigTomlTargetName,
                        "0640"),
        }, false, "xspot", "Exotanium", map[string]string{
		"XSpotAddon": "venkatnamala/xspot-addon:latest",
        }, map[string]string{
                "XSpotAddon": "docker.io",
        }),
	"helm-tiller": NewAddon([]*BinAsset{
		MustBinAsset(addons.HelmTillerAssets,
			"helm-tiller/helm-tiller-dp.tmpl",
			vmpath.GuestAddonsDir,
			"helm-tiller-dp.yaml",
			"0640"),
		MustBinAsset(addons.HelmTillerAssets,
			"helm-tiller/helm-tiller-rbac.tmpl",
			vmpath.GuestAddonsDir,
			"helm-tiller-rbac.yaml",
			"0640"),
		MustBinAsset(addons.HelmTillerAssets,
			"helm-tiller/helm-tiller-svc.tmpl",
			vmpath.GuestAddonsDir,
			"helm-tiller-svc.yaml",
			"0640"),
	}, false, "helm-tiller", "third-party (helm)", map[string]string{
		"Tiller": "helm/tiller:v2.17.0@sha256:4c43eb385032945cad047d2350e4945d913b90b3ab43ee61cecb32a495c6df0f",
	}, map[string]string{
		// GCR is deprecated in helm
		// https://github.com/helm/helm/issues/10004#issuecomment-894478908
		"Tiller": "ghcr.io",
	}),
	"ingress-dns": NewAddon([]*BinAsset{
		MustBinAsset(addons.IngressDNSAssets,
			"ingress-dns/ingress-dns-pod.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"ingress-dns-pod.yaml",
			"0640"),
	}, false, "ingress-dns", "google", map[string]string{
		"IngressDNS": "k8s-minikube/minikube-ingress-dns:0.0.2@sha256:4abe27f9fc03fedab1d655e2020e6b165faf3bf6de1088ce6cf215a75b78f05f",
	}, map[string]string{
		"IngressDNS": "gcr.io",
	}),
	"metallb": NewAddon([]*BinAsset{
		MustBinAsset(addons.MetallbAssets,
			"metallb/metallb.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"metallb.yaml",
			"0640"),
		MustBinAsset(addons.MetallbAssets,
			"metallb/metallb-config.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"metallb-config.yaml",
			"0640"),
	}, false, "metallb", "third-party (metallb)", map[string]string{
		"Speaker":    "metallb/speaker:v0.9.6@sha256:c66585a805bed1a3b829d8fb4a4aab9d87233497244ebff96f1b88f1e7f8f991",
		"Controller": "metallb/controller:v0.9.6@sha256:fbfdb9d3f55976b0ee38f3309d83a4ca703efcf15d6ca7889cd8189142286502",
	}, nil),
	"ambassador": NewAddon([]*BinAsset{
		MustBinAsset(addons.AmbassadorAssets,
			"ambassador/ambassador-operator-crds.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"ambassador-operator-crds.yaml",
			"0640"),
		MustBinAsset(addons.AmbassadorAssets,
			"ambassador/ambassador-operator.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"ambassador-operator.yaml",
			"0640"),
		MustBinAsset(addons.AmbassadorAssets,
			"ambassador/ambassadorinstallation.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"ambassadorinstallation.yaml",
			"0640"),
	}, false, "ambassador", "third-party (ambassador)", map[string]string{
		"AmbassadorOperator": "datawire/ambassador-operator:v1.2.3@sha256:492f33e0828a371aa23331d75c11c251b21499e31287f026269e3f6ec6da34ed",
	}, map[string]string{
		"AmbassadorOperator": "quay.io",
	}),
	"gcp-auth": NewAddon([]*BinAsset{
		MustBinAsset(addons.GcpAuthAssets,
			"gcp-auth/gcp-auth-ns.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"gcp-auth-ns.yaml",
			"0640"),
		MustBinAsset(addons.GcpAuthAssets,
			"gcp-auth/gcp-auth-service.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"gcp-auth-service.yaml",
			"0640"),
		MustBinAsset(addons.GcpAuthAssets,
			"gcp-auth/gcp-auth-webhook.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"gcp-auth-webhook.yaml",
			"0640"),
	}, false, "gcp-auth", "google", map[string]string{
		"KubeWebhookCertgen": "k8s.gcr.io/ingress-nginx/kube-webhook-certgen:v1.0@sha256:f3b6b39a6062328c095337b4cadcefd1612348fdd5190b1dcbcb9b9e90bd8068",
		"GCPAuthWebhook":     "k8s-minikube/gcp-auth-webhook:v0.0.8@sha256:26c7b2454f1c946d7c80839251d939606620f37c2f275be2796c1ffd96c438f6",
	}, map[string]string{
		"GCPAuthWebhook": "gcr.io",
	}),
	"volumesnapshots": NewAddon([]*BinAsset{
		// make sure the order of apply. `csi-hostpath-snapshotclass` must be the first position, because it depends on `snapshot.storage.k8s.io_volumesnapshotclasses`
		// if user disable volumesnapshots addon and delete `csi-hostpath-snapshotclass` after `snapshot.storage.k8s.io_volumesnapshotclasses`, kubernetes will return the error
		MustBinAsset(addons.VolumeSnapshotsAssets,
			"volumesnapshots/csi-hostpath-snapshotclass.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"csi-hostpath-snapshotclass.yaml",
			"0640"),
		MustBinAsset(addons.VolumeSnapshotsAssets,
			"volumesnapshots/snapshot.storage.k8s.io_volumesnapshotclasses.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"snapshot.storage.k8s.io_volumesnapshotclasses.yaml",
			"0640"),
		MustBinAsset(addons.VolumeSnapshotsAssets,
			"volumesnapshots/snapshot.storage.k8s.io_volumesnapshotcontents.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"snapshot.storage.k8s.io_volumesnapshotcontents.yaml",
			"0640"),
		MustBinAsset(addons.VolumeSnapshotsAssets,
			"volumesnapshots/snapshot.storage.k8s.io_volumesnapshots.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"snapshot.storage.k8s.io_volumesnapshots.yaml",
			"0640"),
		MustBinAsset(addons.VolumeSnapshotsAssets,
			"volumesnapshots/rbac-volume-snapshot-controller.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"rbac-volume-snapshot-controller.yaml",
			"0640"),
		MustBinAsset(addons.VolumeSnapshotsAssets,
			"volumesnapshots/volume-snapshot-controller-deployment.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"volume-snapshot-controller-deployment.yaml",
			"0640"),
	}, false, "volumesnapshots", "kubernetes", map[string]string{
		"SnapshotController": "sig-storage/snapshot-controller:v4.0.0@sha256:00fcc441ea9f72899c25eed61d602272a2a58c5f0014332bdcb5ac24acef08e4",
	}, map[string]string{
		"SnapshotController": "k8s.gcr.io",
	}),
	"csi-hostpath-driver": NewAddon([]*BinAsset{
		MustBinAsset(addons.CsiHostpathDriverAssets,
			"csi-hostpath-driver/rbac/rbac-external-attacher.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"rbac-external-attacher.yaml",
			"0640"),
		MustBinAsset(addons.CsiHostpathDriverAssets,
			"csi-hostpath-driver/rbac/rbac-external-health-monitor-agent.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"rbac-external-health-monitor-agent.yaml",
			"0640"),
		MustBinAsset(addons.CsiHostpathDriverAssets,
			"csi-hostpath-driver/rbac/rbac-external-health-monitor-controller.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"rbac-external-health-monitor-controller.yaml",
			"0640"),
		MustBinAsset(addons.CsiHostpathDriverAssets,
			"csi-hostpath-driver/rbac/rbac-external-provisioner.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"rbac-external-provisioner.yaml",
			"0640"),
		MustBinAsset(addons.CsiHostpathDriverAssets,
			"csi-hostpath-driver/rbac/rbac-external-resizer.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"rbac-external-resizer.yaml",
			"0640"),
		MustBinAsset(addons.CsiHostpathDriverAssets,
			"csi-hostpath-driver/rbac/rbac-external-snapshotter.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"rbac-external-snapshotter.yaml",
			"0640"),
		MustBinAsset(addons.CsiHostpathDriverAssets,
			"csi-hostpath-driver/deploy/csi-hostpath-attacher.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"csi-hostpath-attacher.yaml",
			"0640"),
		MustBinAsset(addons.CsiHostpathDriverAssets,
			"csi-hostpath-driver/deploy/csi-hostpath-driverinfo.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"csi-hostpath-driverinfo.yaml",
			"0640"),
		MustBinAsset(addons.CsiHostpathDriverAssets,
			"csi-hostpath-driver/deploy/csi-hostpath-plugin.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"csi-hostpath-plugin.yaml",
			"0640"),
		MustBinAsset(addons.CsiHostpathDriverAssets,
			"csi-hostpath-driver/deploy/csi-hostpath-provisioner.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"csi-hostpath-provisioner.yaml",
			"0640"),
		MustBinAsset(addons.CsiHostpathDriverAssets,
			"csi-hostpath-driver/deploy/csi-hostpath-resizer.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"csi-hostpath-resizer.yaml",
			"0640"),
		MustBinAsset(addons.CsiHostpathDriverAssets,
			"csi-hostpath-driver/deploy/csi-hostpath-snapshotter.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"csi-hostpath-snapshotter.yaml",
			"0640"),
		MustBinAsset(addons.CsiHostpathDriverAssets,
			"csi-hostpath-driver/deploy/csi-hostpath-storageclass.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"csi-hostpath-storageclass.yaml",
			"0640"),
	}, false, "csi-hostpath-driver", "kubernetes", map[string]string{
		"Attacher":              "sig-storage/csi-attacher:v3.1.0@sha256:50c3cfd458fc8e0bf3c8c521eac39172009382fc66dc5044a330d137c6ed0b09",
		"HostMonitorAgent":      "sig-storage/csi-external-health-monitor-agent:v0.2.0@sha256:c20d4a4772599e68944452edfcecc944a1df28c19e94b942d526ca25a522ea02",
		"HostMonitorController": "sig-storage/csi-external-health-monitor-controller:v0.2.0@sha256:14988b598a180cc0282f3f4bc982371baf9a9c9b80878fb385f8ae8bd04ecf16",
		"NodeDriverRegistrar":   "sig-storage/csi-node-driver-registrar:v2.0.1@sha256:e07f914c32f0505e4c470a62a40ee43f84cbf8dc46ff861f31b14457ccbad108",
		"HostPathPlugin":        "sig-storage/hostpathplugin:v1.6.0@sha256:b526bd29630261eceecf2d38c84d4f340a424d57e1e2661111e2649a4663b659",
		"LivenessProbe":         "sig-storage/livenessprobe:v2.2.0@sha256:48da0e4ed7238ad461ea05f68c25921783c37b315f21a5c5a2780157a6460994",
		"Resizer":               "sig-storage/csi-resizer:v1.1.0@sha256:7a5ba58a44e0d749e0767e4e37315bcf6a61f33ce3185c1991848af4db0fb70a",
		"Snapshotter":           "sig-storage/csi-snapshotter:v4.0.0@sha256:51f2dfde5bccac7854b3704689506aeecfb793328427b91115ba253a93e60782",
		"Provisioner":           "sig-storage/csi-provisioner:v2.1.0@sha256:20c828075d1e36f679d6a91e905b0927141eef5e15be0c9a1ca4a6a0ed9313d2",
	}, map[string]string{
		"Attacher":              "k8s.gcr.io",
		"HostMonitorAgent":      "k8s.gcr.io",
		"HostMonitorController": "k8s.gcr.io",
		"NodeDriverRegistrar":   "k8s.gcr.io",
		"HostPathPlugin":        "k8s.gcr.io",
		"LivenessProbe":         "k8s.gcr.io",
		"Resizer":               "k8s.gcr.io",
		"Snapshotter":           "k8s.gcr.io",
		"Provisioner":           "k8s.gcr.io",
	}),
	"portainer": NewAddon([]*BinAsset{
		MustBinAsset(addons.PortainerAssets,
			"portainer/portainer.yaml.tmpl",
			vmpath.GuestAddonsDir,
			"portainer.yaml",
			"0640"),
	}, false, "portainer", "portainer.io", map[string]string{
		"Portainer": "portainer/portainer-ce:latest@sha256:4f126c5114b63e9d1bceb4b368944d14323329a9a0d4e7bb7eb53c9b7435d498",
	}, nil),
}

// parseMapString creates a map based on `str` which is encoded as <key1>=<value1>,<key2>=<value2>,...
func parseMapString(str string) map[string]string {
	mapResult := make(map[string]string)
	if str == "" {
		return mapResult
	}
	for _, pairText := range strings.Split(str, ",") {
		vals := strings.Split(pairText, "=")
		if len(vals) != 2 {
			out.WarningT("Ignoring invalid pair entry {{.pair}}", out.V{"pair": pairText})
			continue
		}
		mapResult[vals[0]] = vals[1]
	}
	return mapResult
}

// mergeMaps creates a map with the union of `sourceMap` and `overrideMap` where collisions take the value of `overrideMap`.
func mergeMaps(sourceMap, overrideMap map[string]string) map[string]string {
	result := make(map[string]string)
	for name, value := range sourceMap {
		result[name] = value
	}
	for name, value := range overrideMap {
		result[name] = value
	}
	return result
}

// filterKeySpace creates a map of the values in `targetMap` where the keys are also in `keySpace`.
func filterKeySpace(keySpace map[string]string, targetMap map[string]string) map[string]string {
	result := make(map[string]string)
	for name := range keySpace {
		if value, ok := targetMap[name]; ok {
			result[name] = value
		}
	}
	return result
}

// overrideDefaults creates a copy of `defaultMap` where `overrideMap` replaces any of its values that `overrideMap` contains.
func overrideDefaults(defaultMap, overrideMap map[string]string) map[string]string {
	return mergeMaps(defaultMap, filterKeySpace(defaultMap, overrideMap))
}

// SelectAndPersistImages selects which images to use based on addon default images, previously persisted images, and newly requested images - which are then persisted for future enables.
func SelectAndPersistImages(addon *Addon, cc *config.ClusterConfig) (images, customRegistries map[string]string, err error) {
	addonDefaultImages := addon.Images
	if addonDefaultImages == nil {
		addonDefaultImages = make(map[string]string)
	}

	// Use previously configured custom images.
	images = overrideDefaults(addonDefaultImages, cc.CustomAddonImages)
	if viper.IsSet(config.AddonImages) {
		// Parse the AddonImages flag if present.
		newImages := parseMapString(viper.GetString(config.AddonImages))
		for name, image := range newImages {
			if image == "" {
				out.WarningT("Ignoring empty custom image {{.name}}", out.V{"name": name})
				delete(newImages, name)
				continue
			}
			if _, ok := addonDefaultImages[name]; !ok {
				out.WarningT("Ignoring unknown custom image {{.name}}", out.V{"name": name})
			}
		}
		// Use newly configured custom images.
		images = overrideDefaults(addonDefaultImages, newImages)
		// Store custom addon images to be written.
		cc.CustomAddonImages = mergeMaps(cc.CustomAddonImages, images)
	}

	// Use previously configured custom registries.
	customRegistries = filterKeySpace(addonDefaultImages, cc.CustomAddonRegistries) // filter by images map because registry map may omit default registry.
	if viper.IsSet(config.AddonRegistries) {
		// Parse the AddonRegistries flag if present.
		customRegistries = parseMapString(viper.GetString(config.AddonRegistries))
		for name := range customRegistries {
			if _, ok := addonDefaultImages[name]; !ok { // check images map because registry map may omitted default registry
				out.WarningT("Ignoring unknown custom registry {{.name}}", out.V{"name": name})
				delete(customRegistries, name)
			}
		}
		// Since registry map may omit default registry, any previously set custom registries for these images must be cleared.
		for name := range addonDefaultImages {
			delete(cc.CustomAddonRegistries, name)
		}
		// Merge newly set registries into custom addon registries to be written.
		cc.CustomAddonRegistries = mergeMaps(cc.CustomAddonRegistries, customRegistries)
	}

	err = nil
	// If images or registries were specified, save the config afterward.
	if viper.IsSet(config.AddonImages) || viper.IsSet(config.AddonRegistries) {
		// Since these values are only set when a user enables an addon, it is safe to refer to the profile name.
		err = config.Write(viper.GetString(config.ProfileName), cc)
		// Whether err is nil or not we still return here.
	}
	return images, customRegistries, err
}

// GenerateTemplateData generates template data for template assets
func GenerateTemplateData(addon *Addon, cfg config.KubernetesConfig, netInfo NetworkInfo, images, customRegistries map[string]string, enable bool) interface{} {

	a := runtime.GOARCH
	// Some legacy docker images still need the -arch suffix
	// for  less common architectures blank suffix for amd64
	ea := ""
	if runtime.GOARCH != "amd64" {
		ea = "-" + runtime.GOARCH
	}

	opts := struct {
		PreOneTwentyKubernetes bool
		Arch                   string
		ExoticArch             string
		ImageRepository        string
		LoadBalancerStartIP    string
		LoadBalancerEndIP      string
		CustomIngressCert      string
		IngressAPIVersion      string
		ContainerRuntime       string
		RegistryAliases        string
		Images                 map[string]string
		Registries             map[string]string
		CustomRegistries       map[string]string
		NetworkInfo            map[string]string
	}{
		PreOneTwentyKubernetes: false,
		Arch:                   a,
		ExoticArch:             ea,
		ImageRepository:        cfg.ImageRepository,
		LoadBalancerStartIP:    cfg.LoadBalancerStartIP,
		LoadBalancerEndIP:      cfg.LoadBalancerEndIP,
		CustomIngressCert:      cfg.CustomIngressCert,
		RegistryAliases:        cfg.RegistryAliases,
		IngressAPIVersion:      "v1", // api version for ingress (eg, "v1beta1"; defaults to "v1" for k8s 1.19+)
		ContainerRuntime:       cfg.ContainerRuntime,
		Images:                 images,
		Registries:             addon.Registries,
		CustomRegistries:       customRegistries,
		NetworkInfo:            make(map[string]string),
	}
	if opts.ImageRepository != "" && !strings.HasSuffix(opts.ImageRepository, "/") {
		opts.ImageRepository += "/"
	}
	if opts.Registries == nil {
		opts.Registries = make(map[string]string)
	}

	// maintain backwards compatibility with k8s < v1.19
	// by using v1beta1 instead of v1 api version for ingress
	v, err := util.ParseKubernetesVersion(cfg.KubernetesVersion)
	if err != nil {
		return errors.Wrap(err, "parsing Kubernetes version")
	}
	if semver.MustParseRange("<1.19.0")(v) {
		opts.IngressAPIVersion = "v1beta1"
	}
	if semver.MustParseRange("<1.20.0")(v) {
		opts.PreOneTwentyKubernetes = true
	}

	// Network info for generating template
	opts.NetworkInfo["ControlPlaneNodeIP"] = netInfo.ControlPlaneNodeIP
	opts.NetworkInfo["ControlPlaneNodePort"] = fmt.Sprint(netInfo.ControlPlaneNodePort)

	// Append postfix "/" to registries
	for k, v := range opts.Registries {
		if v != "" && !strings.HasSuffix(v, "/") {
			opts.Registries[k] = v + "/"
		}
	}

	for k, v := range opts.CustomRegistries {
		if v != "" && !strings.HasSuffix(v, "/") {
			opts.CustomRegistries[k] = v + "/"
		}
	}

	for name, image := range opts.Images {
		if _, ok := opts.Registries[name]; !ok {
			opts.Registries[name] = "" // Avoid nil access when rendering
		}

		if enable {
			if override, ok := opts.CustomRegistries[name]; ok {
				out.Infof("Using image {{.registry}}{{.image}}", out.V{
					"registry": override,
					// removing the SHA from UI
					// SHA example gcr.io/k8s-minikube/gcp-auth-webhook:v0.0.4@sha256:65e9e69022aa7b0eb1e390e1916e3bf67f75ae5c25987f9154ef3b0e8ab8528b
					"image": strings.Split(image, "@")[0],
				})
			} else if opts.ImageRepository != "" {
				out.Infof("Using image {{.registry}}{{.image}} (global image repository)", out.V{
					"registry": opts.ImageRepository,
					"image":    image,
				})
			} else {
				out.Infof("Using image {{.registry}}{{.image}}", out.V{
					"registry": opts.Registries[name],
					"image":    strings.Split(image, "@")[0],
				})
			}
		}
	}
	return opts
}
