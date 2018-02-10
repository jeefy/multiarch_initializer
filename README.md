## Multi-architecture Initializers

 For those weird people that have things like amd64 and aarch64 in the same cluster... keep being weird.

kubectl -n kube-system patch rs/kube-proxy -f kubernetes/kube-proxy-ds.yaml
