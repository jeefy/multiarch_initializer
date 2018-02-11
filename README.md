## Multi-architecture Initializers

 For those weird people that have things like amd64 and aarch64 in the same cluster... keep being weird.

Let's have some fun.

git clone [repo_url]
cd multiarch_initializer
kubectl apply -f .

{"arm":{"kube-proxy":"gcr.io/google_containers/kube-proxy-arm:v1.9.2"}}

{"arm":{"kube-flannel":"quay.io/coreos/flannel:v0.9.1-arm","install-cni":"quay.io/coreos/flannel:v0.9.1-arm"}}

{"arm":{"kubedns":"gcr.io/google_containers/k8s-dns-kube-dns-amd64:1.14.7","dnsmasq":"gcr.io/google_containers/k8s-dns-dnsmasq-nanny-arm:1.14.7","sidecar":"gcr.io/google_containers/k8s-dns-sidecar-arm:1.14.7"}}

kubeadm join --token 03817a.be461afc1dd805ea 192.168.2.130:6443 --discovery-token-ca-cert-hash sha256:da0acd4f579ce20e57850413a7e9e46f40d1b349466c3d867aac813a3429efee
