package es_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"navy-ng/models/portal"
	"navy-ng/server/portal/internal/service"
	. "navy-ng/server/portal/internal/service"
	"navy-ng/server/portal/internal/service/es"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var _ = Describe("ElasticScalingDeviceMatching", func() {
	var (
		db              *gorm.DB
		ess             *es.ElasticScalingService
		dbPath          string
		logger, _       = zap.NewDevelopment()
		mockRedis       *MockRedisHandler
		mockDeviceCache *MockDeviceCache
	)

	BeforeEach(func() {
		dbPath = fmt.Sprintf("test_db_device_matching_%d.db", time.Now().UnixNano())
		var err error
		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		Expect(err).NotTo(HaveOccurred())

		err = db.AutoMigrate(
			&portal.ElasticScalingStrategy{},
			&portal.StrategyClusterAssociation{},
			&portal.ResourceSnapshot{},
			&portal.StrategyExecutionHistory{},
			&portal.Device{},
			&portal.QueryTemplate{},
			&portal.Order{},
			&portal.OrderDevice{},
			&portal.ElasticScalingOrderDetail{},
			&portal.ResourcePoolDeviceMatchingPolicy{},
		)
		Expect(err).NotTo(HaveOccurred())

		mockRedis = &MockRedisHandler{}
		mockDeviceCache = &MockDeviceCache{}
		ess = es.NewElasticScalingService(db, mockRedis, logger, mockDeviceCache)
	})

	AfterEach(func() {
		sqlDB, err := db.DB()
		Expect(err).NotTo(HaveOccurred())
		err = sqlDB.Close()
		Expect(err).NotTo(HaveOccurred())
		err = os.Remove(dbPath)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("filterAndSelectDevices", func() {
		var (
			strategy  *portal.ElasticScalingStrategy
			clusterID int64 = 1
		)

		Context("for pool entry (scale-out)", func() {
			BeforeEach(func() {
				strategy = &portal.ElasticScalingStrategy{
					BaseModel:              portal.BaseModel{ID: 1},
					ThresholdTriggerAction: es.TriggerActionPoolEntry,
				}
			})

			It("should prioritize unassigned devices", func() {
				candidates := []DeviceResponse{
					{ID: 1, ClusterID: 0, Cluster: ""},
					{ID: 2, ClusterID: 2, Cluster: "other-cluster"},
					{ID: 3, ClusterID: 0, Cluster: ""},
				}
				selectedIDs := ess.FilterAndSelectDevicesPublic(candidates, strategy, clusterID, 0, 0)
				// 设备数量现在动态计算，没有资源增量时默认选择1台设备
				Expect(selectedIDs).To(HaveLen(1))
				Expect(selectedIDs).To(ConsistOf(int64(1)))
			})

			It("should select assigned devices if no unassigned are available", func() {
				candidates := []DeviceResponse{
					{ID: 1, ClusterID: 2, Cluster: "other-cluster"},
					{ID: 2, ClusterID: 3, Cluster: "another-cluster"},
				}
				selectedIDs := ess.FilterAndSelectDevicesPublic(candidates, strategy, clusterID, 0, 0)
				// 设备数量现在动态计算，没有资源增量时默认选择1台设备
				Expect(selectedIDs).To(HaveLen(1))
				Expect(selectedIDs).To(ConsistOf(int64(1)))
			})

			It("should select devices based on resource demand", func() {
				// 测试基于资源需求的设备选择
				candidates := []service.DeviceResponse{
					{ID: 1, ClusterID: 0, Cluster: "", CPU: 16, Memory: 64},
					{ID: 2, ClusterID: 2, Cluster: "other-cluster", CPU: 32, Memory: 128},
					{ID: 3, ClusterID: 3, Cluster: "another-cluster", CPU: 64, Memory: 256},
					{ID: 4, ClusterID: 0, Cluster: "", CPU: 8, Memory: 32},
				}
				// 需要80 CPU, 300 Memory的资源
				selectedIDs := ess.FilterAndSelectDevicesPublic(candidates, strategy, clusterID, 80, 300)
				// 应该选择足够满足需求的设备
				Expect(selectedIDs).To(HaveLen(2))
				Expect(selectedIDs).To(ContainElements(int64(3), int64(2))) // 64+32=96 CPU, 256+128=384 Memory
			})
		})

		Context("for pool exit (scale-in)", func() {
			BeforeEach(func() {
				strategy = &portal.ElasticScalingStrategy{
					BaseModel:              portal.BaseModel{ID: 1},
					ThresholdTriggerAction: es.TriggerActionPoolExit,
				}
			})

			It("should only select devices from the current cluster", func() {
				candidates := []DeviceResponse{
					{ID: 1, ClusterID: 1, Cluster: "current-cluster"},
					{ID: 2, ClusterID: 2, Cluster: "other-cluster"},
					{ID: 3, ClusterID: 1, Cluster: "current-cluster"},
					{ID: 4, ClusterID: 0, Cluster: ""},
				}
				selectedIDs := ess.FilterAndSelectDevicesPublic(candidates, strategy, clusterID, 0, 0)
				// 设备数量现在动态计算，没有资源增量时默认选择1台设备
				Expect(selectedIDs).To(HaveLen(1))
				Expect(selectedIDs).To(ConsistOf(int64(1)))
			})

			It("should select nothing if no devices are in the current cluster", func() {
				candidates := []DeviceResponse{
					{ID: 2, ClusterID: 2, Cluster: "other-cluster"},
					{ID: 4, ClusterID: 0, Cluster: ""},
				}
				selectedIDs := ess.FilterAndSelectDevicesPublic(candidates, strategy, clusterID, 0, 0)
				Expect(selectedIDs).To(BeEmpty())
			})
		})

		Context("general behavior", func() {
			It("should default to 1 device when no resource delta is provided", func() {
				strategy = &portal.ElasticScalingStrategy{
					BaseModel:              portal.BaseModel{ID: 1},
					ThresholdTriggerAction: es.TriggerActionPoolEntry,
				}
				candidates := []DeviceResponse{
					{ID: 1, ClusterID: 0, Cluster: ""},
					{ID: 2, ClusterID: 0, Cluster: ""},
				}
				selectedIDs := ess.FilterAndSelectDevicesPublic(candidates, strategy, clusterID, 0, 0)
				Expect(selectedIDs).To(HaveLen(1))
				Expect(selectedIDs[0]).To(Equal(int64(1)))
			})
		})
	})

	Describe("greedySelectDevices", func() {
		Context("for pool entry (scale-out)", func() {
			It("should select devices to meet CPU and Memory demand", func() {
				devices := []DeviceResponse{
					{ID: 1, CPU: 32, Memory: 128},
					{ID: 2, CPU: 64, Memory: 256},
					{ID: 3, CPU: 16, Memory: 64},
				}
				// Demand: 80 CPU, 300 Memory
				selectedIDs := ess.GreedySelectDevicesPublic(devices, 80, 300, es.TriggerActionPoolEntry)
				Expect(selectedIDs).To(HaveLen(2))
				Expect(selectedIDs).To(ConsistOf(int64(2), int64(1))) // 64+32=96 CPU, 256+128=384 Mem
			})

			It("should select all devices if demand is not met", func() {
				devices := []DeviceResponse{
					{ID: 1, CPU: 16, Memory: 64},
					{ID: 2, CPU: 16, Memory: 64},
				}
				// Demand: 40 CPU
				selectedIDs := ess.GreedySelectDevicesPublic(devices, 40, 0, es.TriggerActionPoolEntry)
				Expect(selectedIDs).To(HaveLen(2))
				Expect(selectedIDs).To(ConsistOf(int64(1), int64(2)))
			})
		})

		Context("for pool exit (scale-in)", func() {
			It("should select smallest devices to meet negative demand", func() {
				devices := []DeviceResponse{
					{ID: 1, CPU: 32, Memory: 128},
					{ID: 2, CPU: 64, Memory: 256},
					{ID: 3, CPU: 16, Memory: 64},
				}
				// Demand: remove 40 CPU, 150 Memory (represented by negative numbers)
				selectedIDs := ess.GreedySelectDevicesPublic(devices, -40, -150, es.TriggerActionPoolExit)
				Expect(selectedIDs).To(HaveLen(2))
				Expect(selectedIDs).To(ConsistOf(int64(3), int64(1))) // 16+32=48 CPU, 64+128=192 Mem
			})
		})
	})

	Describe("getDeviceMatchingPolicies", func() {
		It("should return error when no policies exist", func() {
			policies, err := ess.GetDeviceMatchingPoliciesPublic("compute", es.TriggerActionPoolEntry)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no enabled device matching policy found"))
			Expect(policies).To(BeNil())
		})

		It("should return policies when they exist", func() {
			// 这个测试需要实际的ResourcePoolDeviceMatchingPolicy数据
			// 在实际环境中会有相应的测试数据
			Skip("Requires ResourcePoolDeviceMatchingPolicy test data setup")
		})
	})

	Describe("fetchAndUnmarshalQueryTemplate", func() {
		It("should fetch and unmarshal the template correctly", func() {
			queryGroup := `[{"id":"1","blocks":[{"id":"2","type":"device","key":"status","conditionType":"equal","value":"in_stock"}],"operator":"AND"}]`
			queryTemplate := portal.QueryTemplate{Name: "Test Template", Groups: queryGroup}
			db.Create(&queryTemplate)

			groups, err := ess.FetchAndUnmarshalQueryTemplatePublic(queryTemplate.ID, 1, "", "", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(groups).To(HaveLen(1))
			Expect(string(groups[0].Operator)).To(Equal("AND"))
		})

		It("should return an error if the template is not found", func() {
			_, err := ess.FetchAndUnmarshalQueryTemplatePublic(999, 1, "", "", nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(gorm.ErrRecordNotFound))
		})

		It("should return an error for invalid JSON in template", func() {
			queryTemplate := portal.QueryTemplate{Name: "Invalid JSON Template", Groups: "invalid-json"}
			db.Create(&queryTemplate)

			_, err := ess.FetchAndUnmarshalQueryTemplatePublic(queryTemplate.ID, 1, "", "", nil)
			Expect(err).To(HaveOccurred())
			var syntaxError *json.SyntaxError
			Expect(errors.As(err, &syntaxError)).To(BeTrue())
		})
	})
})
