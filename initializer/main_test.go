package main

import (
	"encoding/json"
	"log"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

func createMockPod(containers map[string]string, newImages map[string]map[string]string) *corev1.Pod {
	pod := corev1.Pod{}
	pod.SetNamespace(corev1.NamespaceDefault)
	pod.SetName("MockPod")
	pod.Spec.NodeName = "foo"
	for index, element := range containers {
		container := corev1.Container{
			Name:  index,
			Image: element,
		}
		pod.Spec.Containers = append(pod.Spec.Containers, container)
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, container)
	}
	if newImages != nil {
		annotations := make(map[string]string)
		data, err := json.Marshal(newImages)
		if err != nil {
			log.Println("Error marshalling annotation data")
		}

		annotations[defaultAnnotation] = string(data)
		pod.SetAnnotations(annotations)

		log.Printf("JSON created: %s", string(data))
	}

	//multiarch.initializer.jeefy.net
	initializers := &metav1.Initializers{}
	initializers.Pending = append(initializers.Pending, metav1.Initializer{Name: defaultInitializerName})
	pod.SetInitializers(initializers)

	return &pod
}

func createMockEnvironment(pod *corev1.Pod) (v1.NodeInterface, v1.PodInterface) {
	obj := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "foo"}}
	obj.Status.NodeInfo.Architecture = "arm"
	clientset := fake.NewSimpleClientset(obj, pod)
	fakeNodeClient := clientset.CoreV1().Nodes()
	fakePodClient := clientset.CoreV1().Pods(corev1.NamespaceDefault)

	return fakeNodeClient, fakePodClient
}

func TestIgnoreSingleImage(t *testing.T) {
	containers := make(map[string]string)
	containers["A"] = "A"
	pod := createMockPod(containers, nil)

	fakeNodeClient, fakePodClient := createMockEnvironment(pod)

	pod, err := initializePod(pod, fakeNodeClient, fakePodClient)

	if err != nil {
		t.Errorf("Error initializing pod: %s", err.Error())
	}

	for _, element := range pod.Spec.Containers {
		if element.Image != containers[element.Name] {
			t.Errorf("Image was incorrect. Expected %s, Got %s", containers[element.Name], element.Image)
		}
	}
	for _, element := range pod.Spec.InitContainers {
		if element.Image != containers[element.Name] {
			t.Errorf("Init Image was incorrect. Expected %s, Got %s", containers[element.Name], element.Image)
		}
	}
}

func TestIgnoreMultipleImages(t *testing.T) {
	containers := make(map[string]string)
	containers["A"] = "A"
	containers["B"] = "B"
	pod := createMockPod(containers, nil)

	fakeNodeClient, fakePodClient := createMockEnvironment(pod)

	pod, err := initializePod(pod, fakeNodeClient, fakePodClient)

	if err != nil {
		t.Errorf("Error initializing pod: %s", err.Error())
	}

	for _, element := range pod.Spec.Containers {
		if element.Image != containers[element.Name] {
			t.Errorf("Image was incorrect. Expected %s, Got %s", containers[element.Name], element.Image)
		}
	}
	for _, element := range pod.Spec.InitContainers {
		if element.Image != containers[element.Name] {
			t.Errorf("Init Image was incorrect. Expected %s, Got %s", containers[element.Name], element.Image)
		}
	}
}

func TestSingleImageSuccess(t *testing.T) {
	containers := make(map[string]string)
	containers["A"] = "A"

	newContainers := make(map[string]map[string]string)
	newImages := make(map[string]string)
	newImages["A"] = "arm_A"
	newContainers["arm"] = newImages

	pod := createMockPod(containers, newContainers)

	fakeNodeClient, fakePodClient := createMockEnvironment(pod)

	pod, err := initializePod(pod, fakeNodeClient, fakePodClient)

	if err != nil {
		t.Errorf("Error initializing pod: %s", err.Error())
	}

	for _, element := range pod.Spec.Containers {
		if element.Image != newContainers["arm"][element.Name] {
			t.Errorf("Image was incorrect. Expected %s, Got %s", newContainers["arm"][element.Name], element.Image)
		}
	}
	for _, element := range pod.Spec.InitContainers {
		if element.Image != newContainers["arm"][element.Name] {
			t.Errorf("Init Image was incorrect. Expected %s, Got %s", newContainers["arm"][element.Name], element.Image)
		}
	}
}

func TestMultipleImageSuccess(t *testing.T) {
	containers := make(map[string]string)
	containers["A"] = "A"
	containers["B"] = "B"

	newContainers := make(map[string]map[string]string)
	newImages := make(map[string]string)
	newImages["A"] = "arm_A"
	newImages["B"] = "arm_B"
	newContainers["arm"] = newImages

	pod := createMockPod(containers, newContainers)

	fakeNodeClient, fakePodClient := createMockEnvironment(pod)

	pod, err := initializePod(pod, fakeNodeClient, fakePodClient)

	if err != nil {
		t.Errorf("Error initializing pod: %s", err.Error())
	}

	for _, element := range pod.Spec.Containers {
		if element.Image != newContainers["arm"][element.Name] {
			t.Errorf("Image was incorrect. Expected %s, Got %s", newContainers["arm"][element.Name], element.Image)
		}
	}
	for _, element := range pod.Spec.InitContainers {
		if element.Image != newContainers["arm"][element.Name] {
			//t.Errorf("Init Image was incorrect. Expected %s, Got %s", newContainers["arm"][element.Name], element.Image)
			t.Errorf("Init Image was incorrect. Expected %s, Got %s", newContainers["arm"][element.Name], element.Image)
		}
	}
}

func TestSingleImageNoArch(t *testing.T) {
	containers := make(map[string]string)
	containers["A"] = "A"
	containers["B"] = "B"

	newContainers := make(map[string]map[string]string)
	newImages := make(map[string]string)
	newImages["A"] = "aarch64_A"
	newImages["B"] = "aarch64_B"
	newContainers["aarch64"] = newImages

	pod := createMockPod(containers, newContainers)

	fakeNodeClient, fakePodClient := createMockEnvironment(pod)

	pod, err := initializePod(pod, fakeNodeClient, fakePodClient)

	if err != nil {
		t.Errorf("Error initializing pod: %s", err.Error())
	}

	for _, element := range pod.Spec.Containers {
		if element.Image != containers[element.Name] {
			t.Errorf("Image was incorrect. Expected %s, Got %s", containers[element.Name], element.Image)
		}
	}
	for _, element := range pod.Spec.InitContainers {
		if element.Image != containers[element.Name] {
			t.Errorf("Init Image was incorrect. Expected %s, Got %s", containers[element.Name], element.Image)
		}
	}
}

func TestSingleImageNotSet(t *testing.T) {
	containers := make(map[string]string)
	containers["A"] = "A"

	newContainers := make(map[string]map[string]string)
	newImages := make(map[string]string)
	newImages["B"] = "arm_B"
	newContainers["arm"] = newImages

	pod := createMockPod(containers, newContainers)

	fakeNodeClient, fakePodClient := createMockEnvironment(pod)

	pod, err := initializePod(pod, fakeNodeClient, fakePodClient)

	if err != nil {
		t.Errorf("Error initializing pod: %s", err.Error())
	}

	for _, element := range pod.Spec.Containers {
		if element.Image != containers[element.Name] {
			t.Errorf("Image was incorrect. Expected %s, Got %s", containers[element.Name], element.Image)
		}
	}
	for _, element := range pod.Spec.InitContainers {
		if element.Image != containers[element.Name] {
			t.Errorf("Init Image was incorrect. Expected %s, Got %s", containers[element.Name], element.Image)
		}
	}
}
