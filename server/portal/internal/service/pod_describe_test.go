package service_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"navy-ng/server/portal/internal/service"
)

var _ = Describe("PodDescribeService", func() {
	var (
		testPodDescribeService *service.TestPodDescribeService
		fakeClientset          *fake.Clientset
		ctx                    context.Context
		testNamespace          string
		testPodName            string
		testClusterName        string
	)

	BeforeEach(func() {
		// Setup test variables
		ctx = context.Background()
		testNamespace = "test-namespace"
		testPodName = "test-pod"
		testClusterName = "test-cluster"

		// Create a fake clientset
		fakeClientset = fake.NewSimpleClientset()

		// Initialize the test service with the fake client
		testPodDescribeService = service.NewTestPodDescribeService(fakeClientset)
	})

	Describe("DescribePod", func() {
		var (
			request  *service.PodDescribeRequest
			testPod  *corev1.Pod
			testTime metav1.Time
		)

		BeforeEach(func() {
			// Setup test time
			testTime = metav1.NewTime(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))

			// Create test pod
			testPod = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              testPodName,
					Namespace:         testNamespace,
					CreationTimestamp: testTime,
					Labels: map[string]string{
						"app": "test-app",
					},
					Annotations: map[string]string{
						"annotation-key": "annotation-value",
					},
				},
				Spec: corev1.PodSpec{
					NodeName: "test-node",
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image:latest",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 8080,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources: corev1.ResourceRequirements{},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "test-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "test-config",
									},
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					PodIP: "10.0.0.1",
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:         "test-container",
							Ready:        true,
							RestartCount: 0,
							State: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{
									StartedAt: testTime,
								},
							},
							Image:   "test-image:latest",
							ImageID: "docker-pullable://test-image@sha256:abcdef123456",
						},
					},
					Conditions: []corev1.PodCondition{
						{
							Type:               corev1.PodReady,
							Status:             corev1.ConditionTrue,
							LastProbeTime:      testTime,
							LastTransitionTime: testTime,
						},
					},
					QOSClass: corev1.PodQOSBestEffort,
				},
			}

			// Add the pod to the fake clientset
			_, err := fakeClientset.CoreV1().Pods(testNamespace).Create(ctx, testPod, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Create test events
			testEvent := &corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-event",
					Namespace: testNamespace,
				},
				InvolvedObject: corev1.ObjectReference{
					Kind:      "Pod",
					Name:      testPodName,
					Namespace: testNamespace,
				},
				Type:           "Normal",
				Reason:         "Started",
				Message:        "Started container",
				FirstTimestamp: testTime,
				LastTimestamp:  testTime,
				Count:          1,
				Source: corev1.EventSource{
					Component: "kubelet",
				},
			}

			// Add the event to the fake clientset
			_, err = fakeClientset.CoreV1().Events(testNamespace).Create(ctx, testEvent, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Setup the request
			request = &service.PodDescribeRequest{
				ClusterName: testClusterName,
				Namespace:   testNamespace,
				PodName:     testPodName,
			}

			// Setup the fake client to return our test event when listing events
			fakeClientset.PrependReactor("list", "events", func(action k8stesting.Action) (bool, runtime.Object, error) {
				listAction := action.(k8stesting.ListAction)
				fieldSelector := listAction.GetListRestrictions().Fields.String()
				
				// Only handle the specific field selector for our test pod
				expectedSelector := "involvedObject.name=test-pod,involvedObject.namespace=test-namespace,involvedObject.kind=Pod"
				if fieldSelector == expectedSelector {
					eventList := &corev1.EventList{
						Items: []corev1.Event{*testEvent},
					}
					return true, eventList, nil
				}
				
				// Let other list actions pass through
				return false, nil, nil
			})
		})

		It("should return pod description successfully", func() {
			// Call the method under test
			response, err := testPodDescribeService.DescribePod(ctx, request)

			// Verify the result
			Expect(err).NotTo(HaveOccurred())
			Expect(response).NotTo(BeNil())
			
			// Verify pod basic info
			Expect(response.PodName).To(Equal(testPodName))
			Expect(response.Namespace).To(Equal(testNamespace))
			Expect(response.Status).To(Equal(string(corev1.PodRunning)))
			Expect(response.NodeName).To(Equal("test-node"))
			Expect(response.IP).To(Equal("10.0.0.1"))
			Expect(response.QoS).To(Equal(string(corev1.PodQOSBestEffort)))
			
			// Verify labels and annotations
			Expect(response.Labels).To(HaveKeyWithValue("app", "test-app"))
			Expect(response.Annotations).To(HaveKeyWithValue("annotation-key", "annotation-value"))
			
			// Verify containers
			Expect(response.Containers).To(HaveLen(1))
			container := response.Containers[0]
			Expect(container.Name).To(Equal("test-container"))
			Expect(container.Image).To(Equal("test-image:latest"))
			Expect(container.Ready).To(BeTrue())
			Expect(container.RestartCount).To(Equal(int32(0)))
			
			// Verify container ports
			Expect(container.Ports).To(HaveLen(1))
			port := container.Ports[0]
			Expect(port.Name).To(Equal("http"))
			Expect(port.ContainerPort).To(Equal(int32(8080)))
			Expect(port.Protocol).To(Equal("TCP"))
			
			// Verify events
			Expect(response.Events).To(HaveLen(1))
			event := response.Events[0]
			Expect(event.Type).To(Equal("Normal"))
			Expect(event.Reason).To(Equal("Started"))
			Expect(event.Message).To(Equal("Started container"))
			Expect(event.From).To(Equal("kubelet"))
			
			// Verify conditions
			Expect(response.Conditions).To(HaveLen(1))
			condition := response.Conditions[0]
			Expect(condition.Type).To(Equal("Ready"))
			Expect(condition.Status).To(Equal("True"))
			
			// Verify volumes
			Expect(response.Volumes).To(HaveLen(1))
			volume := response.Volumes[0]
			Expect(volume.Name).To(Equal("test-volume"))
			Expect(volume.Type).To(Equal("ConfigMap"))
			Expect(volume.Source).To(Equal("test-config"))
		})

		It("should return error when pod not found", func() {
			// Change the request to look for a non-existent pod
			request.PodName = "non-existent-pod"
			
			// Call the method under test
			response, err := testPodDescribeService.DescribePod(ctx, request)
			
			// Verify the result
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should handle empty events gracefully", func() {
			// Override the events reactor to return empty list
			fakeClientset.PrependReactor("list", "events", func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, &corev1.EventList{Items: []corev1.Event{}}, nil
			})
			
			// Call the method under test
			response, err := testPodDescribeService.DescribePod(ctx, request)
			
			// Verify the result
			Expect(err).NotTo(HaveOccurred())
			Expect(response).NotTo(BeNil())
			Expect(response.Events).To(HaveLen(0))
		})
	})
})
