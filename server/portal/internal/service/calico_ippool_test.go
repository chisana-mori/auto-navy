package service

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
)

type fakeRESTConfig struct{}

var _ = Describe("CalicoIPPoolService", func() {
	var (
		ctx           context.Context
		fakeDynamic   *dynamicfake.FakeDynamicClient
		calicoService *CalicoIPPoolService
		clusterName   string
	)

	BeforeEach(func() {
		ctx = context.Background()
		fakeDynamic = dynamicfake.NewSimpleDynamicClient(nil)
		clusterName = "test-cluster"
		// NewCalicoIPPoolService expects map[string]*kubernetes.Clientset.
		// We pass an empty map of this type to satisfy the constructor.
		// Actual test interactions for IPPool listing will rely on the injected dynamicClient.
		calicoService = NewCalicoIPPoolService(make(map[string]*kubernetes.Clientset))
		// Ensure dynamicClients map is initialized before use
		if calicoService.dynamicClients == nil {
			calicoService.dynamicClients = make(map[string]dynamic.Interface)
		}

		// 手动注入fake dynamic client（绕过真实的dynamic.NewForConfig）
		// 直接注入 fake dynamic client, 确保 CalicoIPPoolService 内部逻辑能正确使用
		// 若 CalicoIPPoolService 未直接使用 dynamicClients 字段，则此注入可能无效
		// 需要根据 CalicoIPPoolService.GetClusterIPPools 的实际实现来调整 mock 方式
		if calicoService.dynamicClients == nil {
			calicoService.dynamicClients = make(map[string]dynamic.Interface)
		}
		calicoService.dynamicClients[clusterName] = fakeDynamic // fakeDynamic is already dynamic.Interface
	})

	Describe("convertUnstructuredToIPPoolInfo", func() {
		It("should convert a valid unstructured IPPool object", func() {
			u := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":              "test-pool",
						"creationTimestamp": time.Now().Format(time.RFC3339),
						"labels":            map[string]interface{}{"env": "test"},
					},
					"spec": map[string]interface{}{
						"cidr":         "10.0.0.0/24",
						"ipipMode":     "Never",
						"vxlanMode":    "Never",
						"blockSize":    int64(26),
						"natOutgoing":  true,
						"disabled":     false,
						"nodeSelector": "role=node",
					},
				},
			}
			info, err := convertUnstructuredToIPPoolInfo(*u, clusterName)
			Expect(err).To(BeNil())
			Expect(info.Name).To(Equal("test-pool"))
			Expect(info.CIDR).To(Equal("10.0.0.0/24"))
			Expect(info.Labels["env"]).To(Equal("test"))
			Expect(info.ClusterName).To(Equal(clusterName))
		})
	})

	Describe("GetAllIPPools", func() {
		It("should return empty result for empty cluster", func() {
			calicoServiceWithNoClusters := NewCalicoIPPoolService(make(map[string]*kubernetes.Clientset))
			result, err := calicoServiceWithNoClusters.GetAllIPPools(ctx)
			Expect(err).To(BeNil())
			Expect(result).To(HaveLen(0))
		})
	})

	// 你可以继续补充更多针对 GetClusterIPPools、parseNodeSelector 等方法的测试
})
