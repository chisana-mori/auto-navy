package service_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"navy-ng/models/portal"
	"navy-ng/server/portal/internal/service"
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
		ess             *service.ElasticScalingService
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
		)
		Expect(err).NotTo(HaveOccurred())

		mockRedis = &MockRedisHandler{}
		mockDeviceCache = &MockDeviceCache{}
		ess = service.NewElasticScalingService(db, mockRedis, logger, mockDeviceCache)
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
					ThresholdTriggerAction: service.TriggerActionPoolEntry,
					DeviceCount:            2,
				}
			})

			It("should prioritize unassigned devices", func() {
				candidates := []service.DeviceResponse{
					{ID: 1, ClusterID: 0, Cluster: ""},
					{ID: 2, ClusterID: 2, Cluster: "other-cluster"},
					{ID: 3, ClusterID: 0, Cluster: ""},
				}
				selectedIDs := ess.FilterAndSelectDevicesPublic(candidates, strategy, clusterID, 0, 0)
				Expect(selectedIDs).To(HaveLen(2))
				Expect(selectedIDs).To(ConsistOf(int64(1), int64(3)))
			})

			It("should select assigned devices if no unassigned are available", func() {
				candidates := []service.DeviceResponse{
					{ID: 1, ClusterID: 2, Cluster: "other-cluster"},
					{ID: 2, ClusterID: 3, Cluster: "another-cluster"},
				}
				selectedIDs := ess.FilterAndSelectDevicesPublic(candidates, strategy, clusterID, 0, 0)
				Expect(selectedIDs).To(HaveLen(2))
				Expect(selectedIDs).To(ConsistOf(int64(1), int64(2)))
			})

			It("should select a mix when not enough unassigned devices are available", func() {
				strategy.DeviceCount = 3
				candidates := []service.DeviceResponse{
					{ID: 1, ClusterID: 0, Cluster: ""},
					{ID: 2, ClusterID: 2, Cluster: "other-cluster"},
					{ID: 3, ClusterID: 3, Cluster: "another-cluster"},
					{ID: 4, ClusterID: 0, Cluster: ""},
				}
				selectedIDs := ess.FilterAndSelectDevicesPublic(candidates, strategy, clusterID, 0, 0)
				Expect(selectedIDs).To(HaveLen(3))
				Expect(selectedIDs).To(ConsistOf(int64(1), int64(4), int64(2)))
			})
		})

		Context("for pool exit (scale-in)", func() {
			BeforeEach(func() {
				strategy = &portal.ElasticScalingStrategy{
					BaseModel:              portal.BaseModel{ID: 1},
					ThresholdTriggerAction: service.TriggerActionPoolExit,
					DeviceCount:            2,
				}
			})

			It("should only select devices from the current cluster", func() {
				candidates := []service.DeviceResponse{
					{ID: 1, ClusterID: 1, Cluster: "current-cluster"},
					{ID: 2, ClusterID: 2, Cluster: "other-cluster"},
					{ID: 3, ClusterID: 1, Cluster: "current-cluster"},
					{ID: 4, ClusterID: 0, Cluster: ""},
				}
				selectedIDs := ess.FilterAndSelectDevicesPublic(candidates, strategy, clusterID, 0, 0)
				Expect(selectedIDs).To(HaveLen(2))
				Expect(selectedIDs).To(ConsistOf(int64(1), int64(3)))
			})

			It("should select nothing if no devices are in the current cluster", func() {
				candidates := []service.DeviceResponse{
					{ID: 2, ClusterID: 2, Cluster: "other-cluster"},
					{ID: 4, ClusterID: 0, Cluster: ""},
				}
				selectedIDs := ess.FilterAndSelectDevicesPublic(candidates, strategy, clusterID, 0, 0)
				Expect(selectedIDs).To(BeEmpty())
			})
		})

		Context("general behavior", func() {
			It("should default to 1 device if DeviceCount is zero or negative", func() {
				strategy = &portal.ElasticScalingStrategy{
					BaseModel:              portal.BaseModel{ID: 1},
					ThresholdTriggerAction: service.TriggerActionPoolEntry,
					DeviceCount:            0,
				}
				candidates := []service.DeviceResponse{
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
				devices := []service.DeviceResponse{
					{ID: 1, CPU: 32, Memory: 128},
					{ID: 2, CPU: 64, Memory: 256},
					{ID: 3, CPU: 16, Memory: 64},
				}
				// Demand: 80 CPU, 300 Memory
				selectedIDs := ess.GreedySelectDevicesPublic(devices, 80, 300, service.TriggerActionPoolEntry)
				Expect(selectedIDs).To(HaveLen(2))
				Expect(selectedIDs).To(ConsistOf(int64(2), int64(1))) // 64+32=96 CPU, 256+128=384 Mem
			})

			It("should select all devices if demand is not met", func() {
				devices := []service.DeviceResponse{
					{ID: 1, CPU: 16, Memory: 64},
					{ID: 2, CPU: 16, Memory: 64},
				}
				// Demand: 40 CPU
				selectedIDs := ess.GreedySelectDevicesPublic(devices, 40, 0, service.TriggerActionPoolEntry)
				Expect(selectedIDs).To(HaveLen(2))
				Expect(selectedIDs).To(ConsistOf(int64(1), int64(2)))
			})
		})

		Context("for pool exit (scale-in)", func() {
			It("should select smallest devices to meet negative demand", func() {
				devices := []service.DeviceResponse{
					{ID: 1, CPU: 32, Memory: 128},
					{ID: 2, CPU: 64, Memory: 256},
					{ID: 3, CPU: 16, Memory: 64},
				}
				// Demand: remove 40 CPU, 150 Memory (represented by negative numbers)
				selectedIDs := ess.GreedySelectDevicesPublic(devices, -40, -150, service.TriggerActionPoolExit)
				Expect(selectedIDs).To(HaveLen(2))
				Expect(selectedIDs).To(ConsistOf(int64(3), int64(1))) // 16+32=48 CPU, 64+128=192 Mem
			})
		})
	})

	Describe("getQueryTemplateIDFromStrategy", func() {
		It("should return the correct entry template ID", func() {
			strategy := &portal.ElasticScalingStrategy{
				ThresholdTriggerAction: service.TriggerActionPoolEntry,
				EntryQueryTemplateID:   123,
			}
			id, err := ess.GetQueryTemplateIDFromStrategyPublic(strategy)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).To(Equal(int64(123)))
		})

		It("should return the correct exit template ID", func() {
			strategy := &portal.ElasticScalingStrategy{
				ThresholdTriggerAction: service.TriggerActionPoolExit,
				ExitQueryTemplateID:    456,
			}
			id, err := ess.GetQueryTemplateIDFromStrategyPublic(strategy)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).To(Equal(int64(456)))
		})

		It("should return an error if the template ID is not set", func() {
			strategy := &portal.ElasticScalingStrategy{
				ThresholdTriggerAction: service.TriggerActionPoolEntry,
				// EntryQueryTemplateID is 0
			}
			_, err := ess.GetQueryTemplateIDFromStrategyPublic(strategy)
			Expect(err).To(HaveOccurred())
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
