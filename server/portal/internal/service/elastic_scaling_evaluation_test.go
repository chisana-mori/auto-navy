package service_test

import (
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

// MockRedisHandler 是 RedisHandlerInterface 的一个简单 mock 实现
type MockRedisHandler struct{}

func (m *MockRedisHandler) AcquireLock(key, value string, expiration time.Duration) (bool, error) {
	return true, nil // 在测试中总是成功获取锁
}

func (m *MockRedisHandler) Delete(key string) {
	// Mock delete, do nothing
}

func (m *MockRedisHandler) Expire(expiration time.Duration) {
	// Mock expire, do nothing
}

func (m *MockRedisHandler) Get(key string) string {
	return "" // 默认返回空字符串，模拟缓存未命中
}

func (m *MockRedisHandler) SetWithExpireTime(key string, value string, expiration time.Duration) {
	// Mock set, do nothing
}

func (m *MockRedisHandler) ScanKeys(pattern string) ([]string, error) {
	return []string{}, nil // 默认返回空列表
}

// MockDeviceCache 是 DeviceCacheInterface 的一个简单 mock 实现
type MockDeviceCache struct{}

func (m *MockDeviceCache) Get(key string) (interface{}, bool) {
	return nil, false
}

func (m *MockDeviceCache) Set(key string, value interface{}, d time.Duration) {}

func (m *MockDeviceCache) GetDevice(id int64) (*service.DeviceResponse, error) {
	return nil, errors.New("cache miss")
}

func (m *MockDeviceCache) GetDeviceFieldValues(field string) ([]string, error) {
	return nil, errors.New("cache miss")
}

func (m *MockDeviceCache) GetDeviceList(listType string) (*service.DeviceListResponse, error) {
	return nil, errors.New("cache miss")
}

func (m *MockDeviceCache) InvalidateDeviceLists() error {
	// Mock invalidate, do nothing
	return nil
}

func (m *MockDeviceCache) SetDevice(id int64, device *service.DeviceResponse) error {
	// Mock SetDevice, do nothing
	return nil
}

func (m *MockDeviceCache) SetDeviceFieldValues(field string, values []string, isSystem bool) error {
	// Mock SetDeviceFieldValues, do nothing
	return nil
}

func (m *MockDeviceCache) SetDeviceList(listType string, devices *service.DeviceListResponse) error {
	// Mock SetDeviceList, do nothing
	return nil
}

var _ = Describe("ElasticScalingEvaluation", func() {
	var (
		db              *gorm.DB
		ess             *service.ElasticScalingService
		dbPath          string
		logger, _       = zap.NewDevelopment()
		mockRedis       *MockRedisHandler
		mockDeviceCache *MockDeviceCache
	)

	// 在所有测试开始前，设置测试环境
	BeforeEach(func() {
		// 使用临时的SQLite数据库文件
		dbPath = fmt.Sprintf("test_db_%d.db", time.Now().UnixNano())
		var err error
		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		Expect(err).NotTo(HaveOccurred())

		// 自动迁移模型
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
			&portal.K8sNode{},
			&portal.K8sNodeLabel{},
			&portal.LabelManagement{},
			&portal.K8sNodeTaint{},
			&portal.TaintManagement{},
			&portal.DeviceApp{},
		)
		Expect(err).NotTo(HaveOccurred())

		// 初始化 mock
		mockRedis = &MockRedisHandler{}
		mockDeviceCache = &MockDeviceCache{}

		// 初始化服务
		ess = service.NewElasticScalingService(db, mockRedis, logger, mockDeviceCache)
	})

	// 在每个测试结束后，清理环境
	AfterEach(func() {
		// 关闭数据库连接
		sqlDB, err := db.DB()
		Expect(err).NotTo(HaveOccurred())
		err = sqlDB.Close()
		Expect(err).NotTo(HaveOccurred())

		// 删除临时的数据库文件
		err = os.Remove(dbPath)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("EvaluateSnapshots", func() {
		Context("for pool entry (scale-out) strategies", func() {
			It("should trigger when CPU usage is consistently above threshold", func() {
				strategy := &portal.ElasticScalingStrategy{
					CPUThresholdType:       service.ThresholdTypeUsage,
					CPUThresholdValue:      80,
					ThresholdTriggerAction: service.TriggerActionPoolEntry,
					DurationMinutes:        3, // 需要连续3天
				}
				snapshots := []portal.ResourceSnapshot{
					{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -3))}, MaxCpuUsageRatio: 85},
					{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -2))}, MaxCpuUsageRatio: 90},
					{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -1))}, MaxCpuUsageRatio: 88},
				}

				breached, consecutiveDays, _, _ := ess.EvaluateSnapshots(snapshots, strategy)
				Expect(breached).To(BeTrue())
				Expect(consecutiveDays).To(Equal(3))
			})

			It("should not trigger if usage drops below threshold", func() {
				strategy := &portal.ElasticScalingStrategy{
					CPUThresholdType:       service.ThresholdTypeUsage,
					CPUThresholdValue:      80,
					ThresholdTriggerAction: service.TriggerActionPoolEntry,
					DurationMinutes:        3,
				}
				snapshots := []portal.ResourceSnapshot{
					{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -3))}, MaxCpuUsageRatio: 85},
					{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -2))}, MaxCpuUsageRatio: 75}, // 低于阈值
					{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -1))}, MaxCpuUsageRatio: 88},
				}

				breached, consecutiveDays, _, _ := ess.EvaluateSnapshots(snapshots, strategy)
				Expect(breached).To(BeFalse())
				Expect(consecutiveDays).To(Equal(1)) // 只有最后一天满足
			})

			It("should trigger with AND logic if both CPU and Memory are above threshold", func() {
				strategy := &portal.ElasticScalingStrategy{
					CPUThresholdType:       service.ThresholdTypeUsage,
					CPUThresholdValue:      80,
					MemoryThresholdType:    service.ThresholdTypeUsage,
					MemoryThresholdValue:   70,
					ConditionLogic:         service.ConditionLogicAnd,
					ThresholdTriggerAction: service.TriggerActionPoolEntry,
					DurationMinutes:        2,
				}
				snapshots := []portal.ResourceSnapshot{
					{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -2))}, MaxCpuUsageRatio: 85, MaxMemoryUsageRatio: 75},
					{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -1))}, MaxCpuUsageRatio: 90, MaxMemoryUsageRatio: 80},
				}

				breached, consecutiveDays, _, _ := ess.EvaluateSnapshots(snapshots, strategy)
				Expect(breached).To(BeTrue())
				Expect(consecutiveDays).To(Equal(2))
			})

			It("should trigger with OR logic if either CPU or Memory is above threshold", func() {
				strategy := &portal.ElasticScalingStrategy{
					CPUThresholdType:       service.ThresholdTypeUsage,
					CPUThresholdValue:      80,
					MemoryThresholdType:    service.ThresholdTypeUsage,
					MemoryThresholdValue:   70,
					ConditionLogic:         service.ConditionLogicOr,
					ThresholdTriggerAction: service.TriggerActionPoolEntry,
					DurationMinutes:        2,
				}
				snapshots := []portal.ResourceSnapshot{
					{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -2))}, MaxCpuUsageRatio: 85, MaxMemoryUsageRatio: 65}, // CPU满足
					{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -1))}, MaxCpuUsageRatio: 75, MaxMemoryUsageRatio: 75}, // Memory满足
				}

				breached, consecutiveDays, _, _ := ess.EvaluateSnapshots(snapshots, strategy)
				Expect(breached).To(BeTrue())
				Expect(consecutiveDays).To(Equal(2))
			})
		})

		Context("for pool exit (scale-in) strategies", func() {
			It("should trigger when allocated memory is consistently below threshold", func() {
				strategy := &portal.ElasticScalingStrategy{
					MemoryThresholdType:    service.ThresholdTypeAllocated,
					MemoryThresholdValue:   20,
					ThresholdTriggerAction: service.TriggerActionPoolExit,
					DurationMinutes:        2,
				}
				snapshots := []portal.ResourceSnapshot{
					{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -2))}, MemRequest: 15, MemoryCapacity: 100},
					{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -1))}, MemRequest: 10, MemoryCapacity: 100},
				}

				breached, consecutiveDays, _, _ := ess.EvaluateSnapshots(snapshots, strategy)
				Expect(breached).To(BeTrue())
				Expect(consecutiveDays).To(Equal(2))
			})

			It("should not trigger if allocated memory rises above threshold", func() {
				strategy := &portal.ElasticScalingStrategy{
					MemoryThresholdType:    service.ThresholdTypeAllocated,
					MemoryThresholdValue:   20,
					ThresholdTriggerAction: service.TriggerActionPoolExit,
					DurationMinutes:        2,
				}
				snapshots := []portal.ResourceSnapshot{
					{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -2))}, MemRequest: 15, MemoryCapacity: 100},
					{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -1))}, MemRequest: 25, MemoryCapacity: 100}, // 高于阈值
				}

				breached, consecutiveDays, _, _ := ess.EvaluateSnapshots(snapshots, strategy)
				Expect(breached).To(BeFalse())
				Expect(consecutiveDays).To(Equal(1)) // 应返回观察到的最大连续天数
			})
		})

		Context("general behavior", func() {
			It("should return correct consecutive days when breach happens at the start", func() {
				strategy := &portal.ElasticScalingStrategy{
					CPUThresholdType:       service.ThresholdTypeUsage,
					CPUThresholdValue:      80,
					ThresholdTriggerAction: service.TriggerActionPoolEntry,
					DurationMinutes:        2,
				}
				snapshots := []portal.ResourceSnapshot{
					{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -4))}, MaxCpuUsageRatio: 85},
					{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -3))}, MaxCpuUsageRatio: 90},
					{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -2))}, MaxCpuUsageRatio: 70}, // 中断
					{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -1))}, MaxCpuUsageRatio: 75},
				}

				breached, consecutiveDays, _, _ := ess.EvaluateSnapshots(snapshots, strategy)
				Expect(breached).To(BeFalse())
				Expect(consecutiveDays).To(Equal(2)) // 最大连续天数是2
			})

			It("should return zero consecutive days if no breach occurs", func() {
				strategy := &portal.ElasticScalingStrategy{
					CPUThresholdType:       service.ThresholdTypeUsage,
					CPUThresholdValue:      80,
					ThresholdTriggerAction: service.TriggerActionPoolEntry,
					DurationMinutes:        2,
				}
				snapshots := []portal.ResourceSnapshot{
					{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -2))}, MaxCpuUsageRatio: 70},
					{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -1))}, MaxCpuUsageRatio: 75},
				}

				breached, consecutiveDays, _, _ := ess.EvaluateSnapshots(snapshots, strategy)
				Expect(breached).To(BeFalse())
				Expect(consecutiveDays).To(Equal(0))
			})
		})
	})

	Describe("EvaluateStrategies", func() {
		var strategy *portal.ElasticScalingStrategy

		BeforeEach(func() {
			strategy = &portal.ElasticScalingStrategy{
				Name:            "Test Strategy",
				Status:          "enabled",
				CooldownMinutes: 60,
			}
			db.Create(strategy) // GORM will auto-assign the ID
		})

		Context("when strategy is in cooldown period", func() {
			It("should skip evaluation and create no new history", func() {
				// Arrange: Create a recent execution history to trigger cooldown
				history := portal.StrategyExecutionHistory{
					StrategyID:    strategy.ID,
					ExecutionTime: portal.NavyTime(time.Now().Add(-10 * time.Minute)),
					Result:        "order_created",
				}
				db.Create(&history)

				// Act
				err := ess.EvaluateStrategies()
				Expect(err).NotTo(HaveOccurred())

				// Assert: Verify no new history is created
				var count int64
				db.Model(&portal.StrategyExecutionHistory{}).Count(&count)
				Expect(count).To(Equal(int64(1)))
			})
		})

		Context("when strategy has no cluster associations", func() {
			It("should skip evaluation and create no history", func() {
				// Act
				err := ess.EvaluateStrategies()
				Expect(err).NotTo(HaveOccurred())

				// Assert: Verify no history is created
				var count int64
				db.Model(&portal.StrategyExecutionHistory{}).Count(&count)
				Expect(count).To(Equal(int64(0)))
			})
		})

		Context("when threshold is not met", func() {
			It("should record a 'threshold_not_met' history entry", func() {
				// Arrange
				association := portal.StrategyClusterAssociation{StrategyID: strategy.ID, ClusterID: 101}
				db.Create(&association)

				snapshot := portal.ResourceSnapshot{
					BaseModel:        portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -1))},
					ClusterID:        101,
					MaxCpuUsageRatio: 50, // This does not meet the threshold
				}
				db.Create(&snapshot)

				strategy.CPUThresholdType = service.ThresholdTypeUsage
				strategy.CPUThresholdValue = 80
				strategy.ThresholdTriggerAction = service.TriggerActionPoolEntry
				strategy.DurationMinutes = 1
				db.Save(strategy)

				// Act
				err := ess.EvaluateStrategies()
				Expect(err).NotTo(HaveOccurred())

				// Assert
				var history portal.StrategyExecutionHistory
				err = db.First(&history).Error
				Expect(err).NotTo(HaveOccurred())
				Expect(history.Result).To(Equal(service.StrategyExecutionResultFailureThresholdNotMet))
				Expect(history.StrategyID).To(Equal(strategy.ID))
			})
		})

		Context("when no snapshots are found", func() {
			It("should record a 'no_snapshots' history entry", func() {
				// Arrange: Associate with a cluster but provide no snapshots
				association := portal.StrategyClusterAssociation{StrategyID: strategy.ID, ClusterID: 102}
				db.Create(&association)

				// Act
				err := ess.EvaluateStrategies()
				Expect(err).NotTo(HaveOccurred())

				// Assert
				var history portal.StrategyExecutionHistory
				err = db.First(&history).Error
				Expect(err).NotTo(HaveOccurred())
				Expect(history.Result).To(Equal(service.StrategyExecutionResultFailureNoSnapshots))
				Expect(history.StrategyID).To(Equal(strategy.ID))
			})
		})

		Context("when threshold is consistently breached", func() {
			It("should create an order when conditions are met", func() {
				// Arrange
				// 1. Create a device that can be matched
				device := portal.Device{CICode: "test-device-01", Status: "in_stock"}
				db.Create(&device)

				// 2. Create a QueryTemplate that will find this device
				queryGroup := `[{"id":"1","blocks":[{"id":"2","type":"device","key":"status","conditionType":"equal","value":"in_stock"}],"operator":"AND"}]`
				queryTemplate := portal.QueryTemplate{Name: "Find In-Stock Devices", Groups: queryGroup}
				db.Create(&queryTemplate)

				// 3. Setup strategy to use this template and associate it with a cluster
				strategy.EntryQueryTemplateID = queryTemplate.ID
				strategy.DeviceCount = 1
				strategy.CPUThresholdType = service.ThresholdTypeUsage
				strategy.CPUThresholdValue = 80
				strategy.ThresholdTriggerAction = service.TriggerActionPoolEntry
				strategy.DurationMinutes = 2 // Require 2 days
				db.Save(strategy)

				association := portal.StrategyClusterAssociation{StrategyID: strategy.ID, ClusterID: 103}
				db.Create(&association)

				// 4. Setup snapshots that meet the threshold
				snapshots := []portal.ResourceSnapshot{
					{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -2))}, ClusterID: 103, MaxCpuUsageRatio: 90},
					{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -1))}, ClusterID: 103, MaxCpuUsageRatio: 95},
				}
				db.Create(&snapshots)

				// Act
				err := ess.EvaluateStrategies()
				Expect(err).NotTo(HaveOccurred())

				// Assert
				// 1. Verify an order was created
				var order portal.Order
				err = db.First(&order).Error
				Expect(err).NotTo(HaveOccurred())
				Expect(order.Type).To(Equal(portal.OrderTypeElasticScaling))
				Expect(order.CreatedBy).To(Equal(service.SystemAutoCreator))

				// 2. Verify the order detail was created correctly
				var detail portal.ElasticScalingOrderDetail
				err = db.Where("order_id = ?", order.ID).First(&detail).Error
				Expect(err).NotTo(HaveOccurred())
				Expect(detail.StrategyID).To(HaveValue(Equal(strategy.ID)))

				// 2.1. Verify the OrderDevice was created correctly
				var orderDevice portal.OrderDevice
				err = db.Where("order_id = ?", order.ID).First(&orderDevice).Error
				Expect(err).NotTo(HaveOccurred())
				Expect(orderDevice.DeviceID).To(Equal(device.ID))
				Expect(orderDevice.OrderID).To(Equal(order.ID))

				// 3. Verify the execution history was recorded correctly
				var history portal.StrategyExecutionHistory
				err = db.Where("result = ?", service.StrategyExecutionResultOrderCreated).First(&history).Error
				Expect(err).NotTo(HaveOccurred())
				Expect(history.StrategyID).To(Equal(strategy.ID))
				Expect(history.OrderID).To(HaveValue(Equal(order.ID)))
			})

			Context("when threshold is breached but no devices are found", func() {
				It("should create an order with no devices", func() {
					// Arrange
					// 1. No devices are created, so none can be matched.

					// 2. Create a QueryTemplate that will find nothing
					queryGroup := `[{"id":"1","blocks":[{"id":"2","type":"device","key":"status","conditionType":"equal","value":"non_existent_status"}],"operator":"AND"}]`
					queryTemplate := portal.QueryTemplate{Name: "Find Nothing", Groups: queryGroup}
					db.Create(&queryTemplate)

					// 3. Setup strategy to use this template
					strategy.EntryQueryTemplateID = queryTemplate.ID
					strategy.DeviceCount = 1
					strategy.CPUThresholdType = service.ThresholdTypeUsage
					strategy.CPUThresholdValue = 80
					strategy.ThresholdTriggerAction = service.TriggerActionPoolEntry
					strategy.DurationMinutes = 2 // Require 2 days
					db.Save(strategy)

					association := portal.StrategyClusterAssociation{StrategyID: strategy.ID, ClusterID: 104}
					db.Create(&association)

					// 4. Setup snapshots that meet the threshold
					snapshots := []portal.ResourceSnapshot{
						{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -2))}, ClusterID: 104, MaxCpuUsageRatio: 90},
						{BaseModel: portal.BaseModel{CreatedAt: portal.NavyTime(time.Now().AddDate(0, 0, -1))}, ClusterID: 104, MaxCpuUsageRatio: 95},
					}
					db.Create(&snapshots)

					// Act
					err := ess.EvaluateStrategies()
					Expect(err).NotTo(HaveOccurred())

					// Assert
					// 1. Verify an order was created
					var order portal.Order
					err = db.First(&order).Error
					Expect(err).NotTo(HaveOccurred())
					Expect(order.Type).To(Equal(portal.OrderTypeElasticScaling))

					// 2. Verify the order detail was created correctly
					var detail portal.ElasticScalingOrderDetail
					err = db.Where("order_id = ?", order.ID).First(&detail).Error
					Expect(err).NotTo(HaveOccurred())
					Expect(detail.DeviceCount).To(Equal(0))

					// 3. Verify no devices are associated with the order
					var count int64
					db.Model(&portal.OrderDevice{}).Where("order_detail_id = ?", detail.ID).Count(&count)
					Expect(count).To(Equal(int64(0)))

					// 4. Verify the execution history was recorded correctly
					var history portal.StrategyExecutionHistory
					err = db.Where("result = ?", service.StrategyExecutionResultFailureNoDevicesFound).First(&history).Error
					Expect(err).NotTo(HaveOccurred())
					Expect(history.StrategyID).To(Equal(strategy.ID))
				})
			})
		})
	})
})
