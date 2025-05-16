package service

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// NewTestPodDescribeService creates a PodDescribeService for testing with a fake client
func NewTestPodDescribeService(fakeClient *fake.Clientset) *TestPodDescribeService {
	return &TestPodDescribeService{
		fakeClient: fakeClient,
		service: &PodDescribeService{
			clientCache: make(map[string]*kubernetes.Clientset),
		},
	}
}

// TestPodDescribeService is a wrapper around PodDescribeService for testing
type TestPodDescribeService struct {
	service    *PodDescribeService
	fakeClient *fake.Clientset
}

// DescribePod implements the same interface as PodDescribeService.DescribePod but uses the fake client
func (s *TestPodDescribeService) DescribePod(ctx context.Context, request *PodDescribeRequest) (*PodDescribeResponse, error) {
	// Get Pod information using the fake client
	pod, err := s.fakeClient.CoreV1().Pods(request.Namespace).Get(ctx, request.PodName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf(ErrPodNotFoundMsg, request.PodName, request.Namespace)
	}

	// Get Pod events using the fake client
	events, err := s.getPodEvents(ctx, pod)
	if err != nil {
		// Log error but continue processing
		fmt.Printf("Warning: failed to get pod events: %v\n", err)
	}

	// Build response using the service's method
	response := s.service.buildPodDescribeResponse(pod, events)
	return response, nil
}

// getPodEvents gets pod events using the fake client
func (s *TestPodDescribeService) getPodEvents(ctx context.Context, pod *corev1.Pod) ([]corev1.Event, error) {
	fieldSelector := fmt.Sprintf("involvedObject.name=%s,involvedObject.namespace=%s,involvedObject.kind=Pod",
		pod.Name, pod.Namespace)
	events, err := s.fakeClient.CoreV1().Events(pod.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return nil, err
	}
	return events.Items, nil
}
