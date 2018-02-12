## Multi-architecture Initializer

For those weird people that have things like amd64 and aarch64 in the same cluster... keep being weird.

Let's have some fun.

#### Setup

```
git clone [repo_url]
cd multiarch_initializer
kubectl apply -f manifests/multiarch_configmap.yaml
kubectl apply -f manifests/multiarch_deployment.yaml
```
Make sure the pod has been deployed in kube-system (otherwise race conditions may occur)
*Then*
```
kubectl apply -f manifests/multiarch_initializer_config.yaml
```

#### Race Condition?
Yeah, if you have the initializer configured beforehand, the initializer pod won't start. Oops!

#### Configuring Pods

Add the annotation `initializer.kubernetes.io/multiarch` to the pod/template spec. Values will vary based on what it is.

These examples are for a `Raspberry Pi3` with `Raspbian`
```
kube-proxy
{"arm":{"kube-proxy":"gcr.io/google_containers/kube-proxy-arm:v1.9.2"}}

flannel
{"arm":{"kube-flannel":"quay.io/coreos/flannel:v0.9.1-arm","install-cni":"quay.io/coreos/flannel:v0.9.1-arm"}}

kube-dns
{"arm":{"kubedns":"gcr.io/google_containers/k8s-dns-kube-dns-amd64:1.14.7","dnsmasq":"gcr.io/google_containers/k8s-dns-dnsmasq-nanny-arm:1.14.7","sidecar":"gcr.io/google_containers/k8s-dns-sidecar-arm:1.14.7"}}

prometheus-node-exporter
{"arm":{"prometheus-node-exporter":"rycus86/prometheus-node-exporter:armhf"}}
```
