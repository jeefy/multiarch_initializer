package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const (
	defaultAnnotation      = "initializer.jeefy.net/multiarch"
	defaultInitializerName = "multiarch.initializer.jeefy.net"
	defaultNamespace       = "default"
)

var (
	annotation        string
	initializerName   string
	namespace         string
	requireAnnotation bool
)

var node *corev1.Node
var annotationData map[string]map[string]string

type config struct {
	Containers []corev1.Container
	Volumes    []corev1.Volume
}

func main() {
	flag.StringVar(&annotation, "annotation", defaultAnnotation, "The annotation to trigger initialization")
	flag.StringVar(&initializerName, "initializer-name", defaultInitializerName, "The initializer name")
	flag.StringVar(&namespace, "namespace", "default", "The configuration namespace")
	flag.BoolVar(&requireAnnotation, "require-annotation", false, "Require annotation for initialization")
	flag.Parse()

	log.Println("Starting the Kubernetes initializer...")
	log.Printf("Initializer name set to: %s", initializerName)

	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Watch uninitialized Pods in all namespaces.
	restClient := clientset.CoreV1().RESTClient()
	watchlist := cache.NewListWatchFromClient(restClient, "pods", corev1.NamespaceAll, fields.Everything())

	// Wrap the returned watchlist to workaround the inability to include
	// the `IncludeUninitialized` list option when setting up watch clients.
	includeUninitializedWatchlist := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.IncludeUninitialized = true
			return watchlist.List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.IncludeUninitialized = true
			return watchlist.Watch(options)
		},
	}

	resyncPeriod := 30 * time.Second

	_, controller := cache.NewInformer(includeUninitializedWatchlist, &corev1.Pod{}, resyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod := obj.(*corev1.Pod)
				log.Println("New Pod found: " + pod.Name)
				err := initializePod(obj.(*corev1.Pod), clientset)

				if err != nil {
					log.Println(err)
				}
			},
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	log.Println("Shutdown signal received, exiting...")
	close(stop)
}

func initializePod(pod *corev1.Pod, clientset *kubernetes.Clientset) error {
	if pod.ObjectMeta.GetInitializers() != nil {
		pendingInitializers := pod.ObjectMeta.GetInitializers().Pending

		if initializerName == pendingInitializers[0].Name {
			log.Printf("Initializing pod: %s", pod.Name)

			initializedPod := pod.DeepCopy()

			// Remove self from the list of pending Initializers while preserving ordering.
			if len(pendingInitializers) == 1 {
				initializedPod.ObjectMeta.Initializers = nil
			} else {
				initializedPod.ObjectMeta.Initializers.Pending = append(pendingInitializers[:0], pendingInitializers[1:]...)
			}

			if requireAnnotation {
				a := pod.ObjectMeta.GetAnnotations()
				_, ok := a[annotation]
				if !ok {
					log.Printf("Required '%s' annotation missing; skipping multiarch container injection", annotation)
					_, err := clientset.CoreV1().Pods(pod.Namespace).Update(initializedPod)
					if err != nil {
						return err
					}
					return nil
				}

				annotationData = make(map[string]map[string]string)

				err := json.Unmarshal([]byte(a[annotation]), &annotationData)
				if err != nil {
					log.Println("Error unmarshalling annotation data")
					return err
				}

				nodeClient := clientset.CoreV1().Nodes()
				node, err = nodeClient.Get(pod.Spec.NodeName, metav1.GetOptions{})

				if err != nil {
					log.Println("Error getting node " + pod.Spec.NodeName)
					return err
				}

				if node.Status.NodeInfo.Architecture == "amd64" {
					_, err := clientset.CoreV1().Pods(pod.Namespace).Update(initializedPod)
					if err != nil {
						log.Println("Error initializing pod w/ amd64 defaults")
						return err
					}
					return nil
				}
			}

			for index, element := range initializedPod.Spec.Containers {
				if _, ok := annotationData[node.Status.NodeInfo.Architecture]; ok {
					if val, ok := annotationData[node.Status.NodeInfo.Architecture][element.Name]; ok {
						initializedPod.Spec.Containers[index].Image = val
					} else {
						log.Println("Image not set in annotations for " + node.Status.NodeInfo.Architecture + "/" + element.Name)
						initializedPod.Spec.Containers[index].Image = element.Image
					}
				} else {
					log.Println("Architecture '" + node.Status.NodeInfo.Architecture + "' not set in Pod annotation")
					initializedPod.Spec.Containers[index].Image = element.Image
				}
			}

			for index, element := range initializedPod.Spec.InitContainers {
				if _, ok := annotationData[node.Status.NodeInfo.Architecture]; ok {
					if val, ok := annotationData[node.Status.NodeInfo.Architecture][element.Name]; ok {
						initializedPod.Spec.InitContainers[index].Image = val
					} else {
						log.Println("Image not set in annotations for " + node.Status.NodeInfo.Architecture + "/" + element.Name)
						initializedPod.Spec.InitContainers[index].Image = element.Image
					}
				} else {
					log.Println("Architecture '" + node.Status.NodeInfo.Architecture + "' not set in Pod annotation")
					initializedPod.Spec.InitContainers[index].Image = element.Image
				}
			}

			oldData, err := json.Marshal(pod)
			if err != nil {
				return err
			}

			newData, err := json.Marshal(initializedPod)
			if err != nil {
				return err
			}

			patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, corev1.Pod{})
			if err != nil {
				return err
			}

			_, err = clientset.CoreV1().Pods(pod.Namespace).Patch(pod.Name, types.StrategicMergePatchType, patchBytes)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
