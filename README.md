## Multi-architecture Initializers

 For those weird people that have things like amd64 and aarch64 in the same cluster... keep being weird.

Let's have some fun.

git clone [repo_url]
cd multiarch_initializer
kubectl apply -f .

kubectl patch -f kubernetes/kube-proxy-ds.yaml -p '{"spec":{"template":{"metadata":{"annotations":{"initializer.kubernetes.io/multiarch":"{\"arm\":{\"kube-proxy\":\"gcr.io/google_containers/kube-proxy-arm:v1.9.2\"}}"}}}}}'
