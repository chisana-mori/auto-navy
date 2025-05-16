package service

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"        // Import for sync.WaitGroup
	"sync/atomic" // Import for atomic operations
	"time"

	"go.uber.org/zap" // Added zap import
	"github.com/agiledragon/gomonkey/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"navy-ng/models/portal"
)

// MockRedisHandler implements RedisHandlerInterface for testing
type MockRedisHandler struct {
	lockAcquired bool
	lockKey      string
	lockValue    string
}

// NewMockRedisHandler creates a new mock Redis handler
func NewMockRedisHandler() *MockRedisHandler {
	return &MockRedisHandler{
		lockAcquired: true, // Default to success
	}
}

// AcquireLock mocks acquiring a Redis lock
func (m *MockRedisHandler) AcquireLock(key string, value string, expiry time.Duration) (bool, error) {
	m.lockKey = key
	m.lockValue = value
	return m.lockAcquired, nil
}

// Delete mocks deleting a Redis key
func (m *MockRedisHandler) Delete(key string) {
	// Just record the call, no actual implementation needed
}

// Expire mocks setting expiration on Redis keys
func (m *MockRedisHandler) Expire(expiration time.Duration) {
	// Just record the call, no actual implementation needed
}

// SetLockAcquired sets whether the mock should simulate successful lock acquisition
func (m *MockRedisHandler) SetLockAcquired(acquired bool) {
	m.lockAcquired = acquired
}

// Global variables for test suite
var (
	db               *gorm.DB
	service          *ElasticScalingService
	mockRedisHandler *MockRedisHandler
	logger           *zap.Logger
)

var _ = BeforeSuite(func() {
	// 使用sqlite内存数据库进行测试
	var err error
	db, err = gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	Expect(err).NotTo(HaveOccurred())

	// 自动迁移表结构
	err = db.AutoMigrate(
		&portal.ElasticScalingStrategy{},
		&portal.StrategyClusterAssociation{},
		&portal.ResourceSnapshot{},
		&portal.StrategyExecutionHistory{},
		&portal.ElasticScalingOrder{},
		&portal.OrderDevice{},
		&portal.K8sCluster{},
		&portal.Device{},
	)
	Expect(err).NotTo(HaveOccurred())

	// Create a mock Redis handler
	mockRedisHandler = NewMockRedisHandler()

	// Create a logger for tests
	var errLogger error
	logger, errLogger = zap.NewDevelopment() // Or zap.NewExample() or a configured logger
	Expect(errLogger).NotTo(HaveOccurred())

	// Pass the mock Redis handler and logger to the service constructor
	service = NewElasticScalingService(db, mockRedisHandler, logger)
})

var _ = AfterSuite(func() {
	// 清理数据库 - 使用事务来确保原子性
	db.Transaction(func(tx *gorm.DB) error {
		// 获取实际的表名，这些是基于模型定义中的 TableName() 方法
		tablesToClear := []string{
			(portal.ElasticScalingStrategy{}).TableName(),
			(portal.StrategyClusterAssociation{}).TableName(),
			(portal.ResourceSnapshot{}).TableName(),
			(portal.StrategyExecutionHistory{}).TableName(),
			(portal.ElasticScalingOrder{}).TableName(),
			(portal.OrderDevice{}).TableName(),
			(portal.K8sCluster{}).TableName(),
			(portal.Device{}).TableName(),
		}
		for _, table := range tablesToClear {
			// 使用 Unscoped 来确保所有记录都被删除，包括软删除的记录
			if err := tx.Exec("DELETE FROM " + table).Error; err != nil {
				// 如果表不存在，忽略错误
				if !strings.Contains(err.Error(), "no such table") {
					return err
				}
			}
		}
		return nil
	})
})

var _ = Describe("ElasticScalingService", func() {
	BeforeEach(func() {
		// 每个测试前清理数据库表
		db.Transaction(func(tx *gorm.DB) error {
			tablesToClear := []string{
				(portal.ElasticScalingStrategy{}).TableName(),
				(portal.StrategyClusterAssociation{}).TableName(),
				(portal.ResourceSnapshot{}).TableName(),
				(portal.StrategyExecutionHistory{}).TableName(),
				(portal.ElasticScalingOrder{}).TableName(),
				(portal.OrderDevice{}).TableName(),
				(portal.K8sCluster{}).TableName(),
				(portal.Device{}).TableName(),
			}
			for _, table := range tablesToClear {
				if err := tx.Exec("DELETE FROM " + table).Error; err != nil {
					if !strings.Contains(err.Error(), "no such table") {
						return err
					}
				}
			}
			return nil
		})
	})

	Describe("EvaluateStrategies", func() {
		// Test case: No enabled strategies
		Context("when there are no enabled strategies", func() {
			// No setup needed - the database is empty by default

			It("should return nil and not create any orders", func() {
				// Test the method
				err := service.EvaluateStrategies()
				Expect(err).NotTo(HaveOccurred())

				// Verify no orders were created
				var orderCount int64
				db.Model(&portal.ElasticScalingOrder{}).Count(&orderCount)
				Expect(orderCount).To(BeZero())
			})
		})

		// Test case: Enabled strategy with no associated clusters
		Context("when there is an enabled strategy with no associated clusters", func() {
			BeforeEach(func() {
				strategy := portal.ElasticScalingStrategy{
					Name:                   "Test Strategy",
					Status:                 "enabled",
					ThresholdTriggerAction: "pool_entry",
					CPUThresholdValue:      80,
					CPUThresholdType:       "usage",
					DeviceCount:            1,
					DurationMinutes:        5,
					CooldownMinutes:        10,
					CreatedBy:              "test",
				}
				err := db.Create(&strategy).Error
				Expect(err).NotTo(HaveOccurred())
			})

			It("should not create any orders", func() {
				// Test the method
				err := service.EvaluateStrategies()
				Expect(err).NotTo(HaveOccurred())

				// Verify no orders were created
				var orderCount int64
				db.Model(&portal.ElasticScalingOrder{}).Count(&orderCount)
				Expect(orderCount).To(BeZero())

				var historyCount int64
				db.Model(&portal.StrategyExecutionHistory{}).Count(&historyCount)
				Expect(historyCount).To(BeZero()) // No history should be recorded if no clusters are associated
			})
		})

		// Test case: Enabled strategy with associated clusters but no resource snapshots
		Context("when there is an enabled strategy with associated clusters but no resource snapshots", func() {
			var clusterID int64
			var strategyID int64

			BeforeEach(func() {
				// Create a cluster
				cluster := portal.K8sCluster{
					ClusterName: "Test Cluster",
					ClusterID:   "test-cluster-id",
					Status:      "active",
				}
				err := db.Create(&cluster).Error
				Expect(err).NotTo(HaveOccurred())
				clusterID = cluster.ID

				// Create a strategy
				strategy := portal.ElasticScalingStrategy{
					Name:                   "Test Strategy",
					Status:                 "enabled",
					ThresholdTriggerAction: "pool_entry",
					CPUThresholdValue:      80,
					CPUThresholdType:       "usage",
					DeviceCount:            1,
					DurationMinutes:        5,
					CooldownMinutes:        10,
					CreatedBy:              "test",
				}
				err = db.Create(&strategy).Error
				Expect(err).NotTo(HaveOccurred())
				strategyID = strategy.ID

				// Associate cluster with strategy
				association := portal.StrategyClusterAssociation{
					StrategyID: strategyID,
					ClusterID:  clusterID,
				}
				err = db.Create(&association).Error
				Expect(err).NotTo(HaveOccurred())
			})

			It("should record skipped execution history for each resource type", func() {
				err := service.EvaluateStrategies()
				Expect(err).NotTo(HaveOccurred())

				var history []portal.StrategyExecutionHistory
				db.Where("strategy_id = ?", strategyID).Find(&history)
				Expect(len(history)).To(BeNumerically(">", 0))
				
				for _, h := range history {
					Expect(h.Result).To(Equal("skipped"))
					Expect(h.Reason).To(ContainSubstring("没有资源类型"))
				}

				var orderCount int64
				db.Model(&portal.ElasticScalingOrder{}).Count(&orderCount)
				Expect(orderCount).To(BeZero())
			})
		})

		// Test case: Enabled strategy with associated clusters and resource snapshots, but conditions not met
		Context("when there is an enabled strategy with associated clusters and resource snapshots, but conditions not met", func() {
			var strategyID int64
			var clusterID int64

			BeforeEach(func() {
				// Create a cluster
				cluster := portal.K8sCluster{
					ClusterName: "Test Cluster",
					ClusterID:   "test-cluster-id-2",
					Status:      "active",
				}
				err := db.Create(&cluster).Error
				Expect(err).NotTo(HaveOccurred())
				clusterID = cluster.ID

				// Create a strategy
				strategy := portal.ElasticScalingStrategy{
					Name:                   "Test Strategy",
					Status:                 "enabled",
					ThresholdTriggerAction: "pool_entry",
					CPUThresholdValue:      80, // Threshold is 80%
					CPUThresholdType:       "usage",
					DeviceCount:            1,
					DurationMinutes:        5,
					CooldownMinutes:        10,
					CreatedBy:              "test",
					ResourceTypes:          "total",
				}
				err = db.Create(&strategy).Error
				Expect(err).NotTo(HaveOccurred())
				strategyID = strategy.ID

				// Associate cluster with strategy
				association := portal.StrategyClusterAssociation{
					StrategyID: strategyID,
					ClusterID:  clusterID,
				}
				err = db.Create(&association).Error
				Expect(err).NotTo(HaveOccurred())

				// Create a resource snapshot with CPU usage below threshold (70% < 80%)
				snapshot := portal.ResourceSnapshot{
					ClusterID:         uint(clusterID),
					ResourceType:      "total",
					ResourcePool:      "compute",
					MaxCpuUsageRatio:  70, // Below threshold
					CpuCapacity:       100,
					CpuRequest:        60,
					MemoryCapacity:    100,
					MemRequest:        50,
				}
				err = db.Create(&snapshot).Error
				Expect(err).NotTo(HaveOccurred())
			})

			It("should not create any orders and record monitoring history", func() {
				// Test the method
				err := service.EvaluateStrategies()
				Expect(err).NotTo(HaveOccurred())

				// Verify no orders were created
				var orderCount int64
				db.Model(&portal.ElasticScalingOrder{}).Count(&orderCount)
				Expect(orderCount).To(BeZero())

				// Verify history was recorded
				var history []portal.StrategyExecutionHistory
				db.Where("strategy_id = ?", strategyID).Find(&history)
				Expect(len(history)).To(BeNumerically(">", 0))
				
				// Verify the history shows monitoring but no trigger
				for _, h := range history {
					Expect(h.Result).To(Equal("skipped"))
					Expect(h.OrderID).To(BeNil())
				}
			})
		})

		// Test case: Enabled strategy with associated clusters and resource snapshots, conditions met but not for duration
		Context("when conditions are met but not for the required duration", func() {
			var strategyID int64
			var clusterID int64

			BeforeEach(func() {
				// Create a cluster
				cluster := portal.K8sCluster{
					ClusterName: "Test Cluster",
					ClusterID:   "test-cluster-id-3",
					Status:      "active",
				}
				err := db.Create(&cluster).Error
				Expect(err).NotTo(HaveOccurred())
				clusterID = cluster.ID

				// Create a strategy with 5 minutes duration
				strategy := portal.ElasticScalingStrategy{
					Name:                   "Test Strategy",
					Status:                 "enabled",
					ThresholdTriggerAction: "pool_entry",
					CPUThresholdValue:      80,
					CPUThresholdType:       "usage",
					DeviceCount:            1,
					DurationMinutes:        5, // 5 minutes required
					CooldownMinutes:        10,
					CreatedBy:              "test",
					ResourceTypes:          "total",
				}
				err = db.Create(&strategy).Error
				Expect(err).NotTo(HaveOccurred())
				strategyID = strategy.ID

				// Associate cluster with strategy
				association := portal.StrategyClusterAssociation{
					StrategyID: strategyID,
					ClusterID:  clusterID,
				}
				err = db.Create(&association).Error
				Expect(err).NotTo(HaveOccurred())

				// Create a resource snapshot with CPU usage above threshold (90% > 80%)
				// But only 3 minutes ago (less than the required 5 minutes)
				snapshot := portal.ResourceSnapshot{
					ClusterID:         uint(clusterID),
					ResourceType:      "total",
					ResourcePool:      "compute",
					MaxCpuUsageRatio:  90, // Above threshold
					CpuCapacity:       100,
					CpuRequest:        80,
					MemoryCapacity:    100,
					MemRequest:        70,
				}
				err = db.Create(&snapshot).Error
				Expect(err).NotTo(HaveOccurred())

				// Create another snapshot with CPU usage above threshold, but more recent
				snapshot2 := portal.ResourceSnapshot{
					ClusterID:         uint(clusterID),
					ResourceType:      "total",
					ResourcePool:      "compute",
					MaxCpuUsageRatio:  95, // Above threshold
					CpuCapacity:       100,
					CpuRequest:        85,
					MemoryCapacity:    100,
					MemRequest:        75,
				}
				err = db.Create(&snapshot2).Error
				Expect(err).NotTo(HaveOccurred())
			})

			It("should not create any orders but record that conditions are being monitored", func() {
				// Test the method
				err := service.EvaluateStrategies()
				Expect(err).NotTo(HaveOccurred())

				// Verify no orders were created
				var orderCount int64
				db.Model(&portal.ElasticScalingOrder{}).Count(&orderCount)
				Expect(orderCount).To(BeZero())

				// Verify history was recorded
				var history []portal.StrategyExecutionHistory
				db.Where("strategy_id = ?", strategyID).Find(&history)
				Expect(len(history)).To(BeNumerically(">", 0))
				
				// Verify the history shows monitoring and mentions duration
				for _, h := range history {
					Expect(h.Result).To(Equal("skipped"))
					Expect(h.OrderID).To(BeNil())
				}
			})
		})

		// Test case: Enabled strategy with associated clusters and resource snapshots, conditions met for duration, pool_entry action
		Context("when conditions are met for the required duration with pool_entry action", func() {
			var strategyID int64
			var clusterID int64

			BeforeEach(func() {
				// Create a cluster
				cluster := portal.K8sCluster{
					ClusterName: "Test Cluster",
					ClusterID:   "test-cluster-id-4",
					Status:      "active",
				}
				err := db.Create(&cluster).Error
				Expect(err).NotTo(HaveOccurred())
				clusterID = cluster.ID

				// Create a strategy with 5 minutes duration and pool_entry action
				strategy := portal.ElasticScalingStrategy{
					Name:                   "Test Strategy",
					Status:                 "enabled",
					ThresholdTriggerAction: "pool_entry", // Add devices to pool
					CPUThresholdValue:      80,
					CPUThresholdType:       "usage",
					DeviceCount:            2, // Add 2 devices
					DurationMinutes:        5,
					CooldownMinutes:        10,
					CreatedBy:              "test",
					ResourceTypes:          "total",
				}
				err = db.Create(&strategy).Error
				Expect(err).NotTo(HaveOccurred())
				strategyID = strategy.ID

				// Associate cluster with strategy
				association := portal.StrategyClusterAssociation{
					StrategyID: strategyID,
					ClusterID:  clusterID,
				}
				err = db.Create(&association).Error
				Expect(err).NotTo(HaveOccurred())

				// Create resource snapshots with CPU usage above threshold for the duration
				// We'll create snapshots for each minute in the duration period
				for i := 0; i <= strategy.DurationMinutes; i++ {
					snapshot := portal.ResourceSnapshot{
						ClusterID:         uint(clusterID),
						ResourceType:      "total",
						ResourcePool:      "compute",
						MaxCpuUsageRatio:  85 + float64(i), // Above threshold
						CpuCapacity:       100,
						CpuRequest:        80 + float64(i),
						MemoryCapacity:    100,
						MemRequest:        70,
					}
					err := db.Create(&snapshot).Error
					Expect(err).NotTo(HaveOccurred())
				}
			})

			It("should create a pool_entry order", func() {
				// Mock the Redis lock to be acquired successfully
				patch := gomonkey.ApplyMethod(reflect.TypeOf(service.RedisHandler), "AcquireLock", 
					func(_ *redis.RedisHandler, _ string) (bool, error) {
						return true, nil
					})
				defer patch.Reset()

				// Test the method
				err := service.EvaluateStrategies()
				Expect(err).NotTo(HaveOccurred())

				// Verify an order was created
				var orders []portal.ElasticScalingOrder
				db.Where("strategy_id = ?", strategyID).Find(&orders)
				Expect(len(orders)).To(Equal(1))

				// Verify order details
				Expect(orders[0].ActionType).To(Equal("pool_entry"))
				Expect(orders[0].ClusterID).To(Equal(clusterID))
				Expect(*orders[0].StrategyID).To(Equal(strategyID))
				Expect(orders[0].DeviceCount).To(Equal(2))
				Expect(orders[0].Status).To(Equal("pending"))

				// Verify history was recorded
				var history []portal.StrategyExecutionHistory
				db.Where("strategy_id = ? AND result = ?", strategyID, "order_created").Find(&history)
				Expect(len(history)).To(Equal(1))
				Expect(*history[0].OrderID).To(Equal(orders[0].ID))
			})
		})

		// Test case: Enabled strategy with associated clusters and resource snapshots, conditions met for duration, pool_exit action
		Context("when conditions are met for the required duration with pool_exit action", func() {
			var strategyID int64
			var clusterID int64

			BeforeEach(func() {
				// Create a cluster
				cluster := portal.K8sCluster{
					ClusterName: "Test Cluster",
					ClusterID:   "test-cluster-id-5",
					Status:      "active",
				}
				err := db.Create(&cluster).Error
				Expect(err).NotTo(HaveOccurred())
				clusterID = cluster.ID

				// Create a strategy with 5 minutes duration and pool_exit action
				strategy := portal.ElasticScalingStrategy{
					Name:                   "Test Strategy",
					Status:                 "enabled",
					ThresholdTriggerAction: "pool_exit", // Remove devices from pool
					CPUThresholdValue:      20, // Low CPU threshold for exit
					CPUThresholdType:       "usage",
					DeviceCount:            1, // Remove 1 device
					DurationMinutes:        5,
					CooldownMinutes:        10,
					CreatedBy:              "test",
					ResourceTypes:          "total",
				}
				err = db.Create(&strategy).Error
				Expect(err).NotTo(HaveOccurred())
				strategyID = strategy.ID

				// Associate cluster with strategy
				association := portal.StrategyClusterAssociation{
					StrategyID: strategyID,
					ClusterID:  clusterID,
				}
				err = db.Create(&association).Error
				Expect(err).NotTo(HaveOccurred())

				// Create resource snapshots with CPU usage below threshold for the duration
				// We'll create snapshots for each minute in the duration period
				for i := 0; i <= strategy.DurationMinutes; i++ {
					snapshot := portal.ResourceSnapshot{
						ClusterID:         uint(clusterID),
						ResourceType:      "total",
						ResourcePool:      "compute",
						MaxCpuUsageRatio:  15 - float64(i), // Below threshold
						CpuCapacity:       100,
						CpuRequest:        10 - float64(i),
						MemoryCapacity:    100,
						MemRequest:        30,
					}
					err := db.Create(&snapshot).Error
					Expect(err).NotTo(HaveOccurred())
				}

				// Create a device for the pool_exit action
				device := portal.Device{
					CICode:    "test-device-id",
					ClusterID: int(clusterID),
					Status:    "active",
				}
				err = db.Create(&device).Error
				Expect(err).NotTo(HaveOccurred())
			})

			It("should create a pool_exit order", func() {
				// Mock the Redis lock to be acquired successfully
				patch := gomonkey.ApplyMethod(reflect.TypeOf(service.RedisHandler), "AcquireLock", 
					func(_ *redis.RedisHandler, _ string) (bool, error) {
						return true, nil
					})
				defer patch.Reset()

				// Test the method
				err := service.EvaluateStrategies()
				Expect(err).NotTo(HaveOccurred())

				// Verify an order was created
				var orders []portal.ElasticScalingOrder
				db.Where("strategy_id = ?", strategyID).Find(&orders)
				Expect(len(orders)).To(Equal(1))

				// Verify order details
				Expect(orders[0].ActionType).To(Equal("pool_exit"))
				Expect(orders[0].ClusterID).To(Equal(clusterID))
				Expect(*orders[0].StrategyID).To(Equal(strategyID))
				Expect(orders[0].DeviceCount).To(Equal(1))
				Expect(orders[0].Status).To(Equal("pending"))

				// Verify history was recorded
				var history []portal.StrategyExecutionHistory
				db.Where("strategy_id = ? AND result = ?", strategyID, "order_created").Find(&history)
				Expect(len(history)).To(Equal(1))
				Expect(*history[0].OrderID).To(Equal(orders[0].ID))
			})
		})

		// Test case: Enabled strategy in cooldown period
		Context("when the strategy is in cooldown period", func() {
			var strategyID int64
			var clusterID int64

			BeforeEach(func() {
				// Create a cluster
				cluster := portal.K8sCluster{
					ClusterName: "Test Cluster",
					ClusterID:   "test-cluster-id-6",
					Status:      "active",
				}
				err := db.Create(&cluster).Error
				Expect(err).NotTo(HaveOccurred())
				clusterID = cluster.ID

				// Create a strategy with 10 minutes cooldown
				strategy := portal.ElasticScalingStrategy{
					Name:                   "Test Strategy",
					Status:                 "enabled",
					ThresholdTriggerAction: "pool_entry",
					CPUThresholdValue:      80,
					CPUThresholdType:       "usage",
					DeviceCount:            1,
					DurationMinutes:        5,
					CooldownMinutes:        10, // 10 minutes cooldown
					CreatedBy:              "test",
					ResourceTypes:          "total",
				}
				err = db.Create(&strategy).Error
				Expect(err).NotTo(HaveOccurred())
				strategyID = strategy.ID

				// Associate cluster with strategy
				association := portal.StrategyClusterAssociation{
					StrategyID: strategyID,
					ClusterID:  clusterID,
				}
				err = db.Create(&association).Error
				Expect(err).NotTo(HaveOccurred())

				// Create a resource snapshot with CPU usage above threshold
				snapshot := portal.ResourceSnapshot{
					ClusterID:         uint(clusterID),
					ResourceType:      "total",
					ResourcePool:      "compute",
					MaxCpuUsageRatio:  90, // Above threshold
					CpuCapacity:       100,
					CpuRequest:        80,
					MemoryCapacity:    100,
					MemRequest:        70,
				}
				err = db.Create(&snapshot).Error
				Expect(err).NotTo(HaveOccurred())

				// Create a recent order to put the strategy in cooldown
				order := portal.ElasticScalingOrder{
					ActionType:  "pool_entry",
					ClusterID:   clusterID,
					StrategyID:  &strategyID,
					DeviceCount: 1,
					Status:      "success",
				}
				// Set the CreatedAt field using the BaseModel
				order.BaseModel.CreatedAt = time.Now().Add(-5 * time.Minute) // 5 minutes ago (still in 10 min cooldown)
				err = db.Create(&order).Error
				Expect(err).NotTo(HaveOccurred())
			})

			It("should not create any orders and record skipped execution history", func() {
				// Test the method
				err := service.EvaluateStrategies()
				Expect(err).NotTo(HaveOccurred())

				// Verify no new orders were created
				var orderCount int64
				db.Model(&portal.ElasticScalingOrder{}).Where("created_at > ?", time.Now().Add(-1*time.Minute)).Count(&orderCount)
				Expect(orderCount).To(BeZero())

				// Verify history was recorded
				var history []portal.StrategyExecutionHistory
				db.Where("strategy_id = ?", strategyID).Find(&history)
				Expect(len(history)).To(BeNumerically(">", 0))
				
				// Verify the history shows skipped due to cooldown
				for _, h := range history {
					Expect(h.Result).To(Equal("skipped"))
					Expect(h.Reason).To(ContainSubstring("cooldown"))
					Expect(h.OrderID).To(BeNil())
				}
			})
		})
	})
	})

