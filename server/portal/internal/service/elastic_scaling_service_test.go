package service

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"navy-ng/models/portal"
)

// MockRedisHandler is a mock implementation of RedisHandlerInterface
type MockRedisHandler struct {
	mock.Mock
}

func (m *MockRedisHandler) AcquireLock(key string, value string, expiry time.Duration) (bool, error) {
	args := m.Called(key, value, expiry)
	return args.Bool(0), args.Error(1)
}

func (m *MockRedisHandler) Delete(key string) {
	m.Called(key)
}

func (m *MockRedisHandler) Expire(expiration time.Duration) {
	m.Called(expiration)
}

// MockDeviceCache is a mock implementation of DeviceCache
type MockDeviceCache struct {
	mock.Mock
}

func (m *MockDeviceCache) GetDeviceList(queryHash string) (*DeviceListResponse, error) {
	args := m.Called(queryHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DeviceListResponse), args.Error(1)
}

func (m *MockDeviceCache) SetDeviceList(queryHash string, response *DeviceListResponse) error {
	args := m.Called(queryHash, response)
	return args.Error(0)
}

func (m *MockDeviceCache) InvalidateDeviceLists() {
	m.Called()
}

func (m *MockDeviceCache) GetDevice(deviceID int64) (*DeviceResponse, error) {
	args := m.Called(deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DeviceResponse), args.Error(1)
}

func (m *MockDeviceCache) SetDevice(deviceID int64, device *DeviceResponse) error {
	args := m.Called(deviceID, device)
	return args.Error(0)
}

func (m *MockDeviceCache) GetDeviceFieldValues(field string) ([]string, error) {
	args := m.Called(field)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockDeviceCache) SetDeviceFieldValues(field string, values []string, isLabelField bool) error {
	args := m.Called(field, values, isLabelField)
	return args.Error(0)
}


type ElasticScalingServiceTestSuite struct {
	suite.Suite
	service      *ElasticScalingService
	db           *gorm.DB
	sqlMock      sqlmock.Sqlmock
	redisHandler *MockRedisHandler
	deviceCache  *MockDeviceCache
	logger       *zap.Logger
}

func (s *ElasticScalingServiceTestSuite) SetupTest() {
	var err error
	s.logger = zap.NewNop() // Use No-Op logger for tests

	// Setup sqlmock
	mockDb, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp)) // Use regexp matching
	assert.NoError(s.T(), err)

	// Setup GORM with the mocked SQL driver
	dialector := mysql.New(mysql.Config{
		Conn:       mockDb,
		SkipInitializeWithVersion: true,
	})
	s.db, err = gorm.Open(dialector, &gorm.Config{
		Logger: zap.NewNop().Sugar(), // Disable GORM logging for tests
	})
	assert.NoError(s.T(), err)
	s.sqlMock = mock

	s.redisHandler = new(MockRedisHandler)
	s.deviceCache = new(MockDeviceCache)

	s.service = NewElasticScalingService(s.db, s.redisHandler, s.logger, s.deviceCache)
}

func TestElasticScalingServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ElasticScalingServiceTestSuite))
}

// --- Test Cases Start Here ---

func (s *ElasticScalingServiceTestSuite) TestEvaluateStrategy_CooldownPeriod() {
	strategy := &portal.ElasticScalingStrategy{
		ID:              1,
		Name:            "TestCooldown",
		Status:          StrategyStatusEnabled,
		CooldownMinutes: 60,
		// ... other necessary fields
	}

	// Mock Redis lock acquisition
	s.redisHandler.On("AcquireLock", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
	s.redisHandler.On("Delete", mock.Anything).Return()

	// Mock DB to return a recent execution history
	expectedHistoryTime := time.Now().Add(-30 * time.Minute) // 30 minutes ago, within 60 min cooldown
	rows := sqlmock.NewRows([]string{"id", "strategy_id", "execution_time", "result"}).
		AddRow(1, strategy.ID, expectedHistoryTime, StrategyExecutionResultOrderCreated)
	
	s.sqlMock.ExpectQuery("SELECT \\* FROM `strategy_execution_histories` WHERE strategy_id = \\? AND result = \\? ORDER BY execution_time DESC LIMIT 1").
		WithArgs(strategy.ID, StrategyExecutionResultOrderCreated).
		WillReturnRows(rows)

	err := s.service.evaluateStrategy(strategy)
	assert.NoError(s.T(), err)

	// Assert that no further DB calls for snapshots or associations happened beyond the history check
	s.sqlMock.ExpectationsWereMet() // Verifies only the history query was made
}


func (s *ElasticScalingServiceTestSuite) TestEvaluateStrategy_NoAssociations() {
	strategy := &portal.ElasticScalingStrategy{
		ID:     1,
		Name:   "TestNoAssociations",
		Status: StrategyStatusEnabled,
		// ... other necessary fields for cooldown check to pass (or mock it empty)
		CooldownMinutes: 60,
	}

	s.redisHandler.On("AcquireLock", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
	s.redisHandler.On("Delete", mock.Anything).Return()

	// Mock DB for cooldown check (no recent history)
	s.sqlMock.ExpectQuery("SELECT \\* FROM `strategy_execution_histories` WHERE strategy_id = \\? AND result = \\? ORDER BY execution_time DESC LIMIT 1").
		WithArgs(strategy.ID, StrategyExecutionResultOrderCreated).
		WillReturnError(gorm.ErrRecordNotFound)
	
	// Mock DB for associations (return empty)
	s.sqlMock.ExpectQuery("SELECT \\* FROM `strategy_cluster_associations` WHERE strategy_id = \\?").
		WithArgs(strategy.ID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "strategy_id", "cluster_id"})) // No rows

	err := s.service.evaluateStrategy(strategy)
	assert.NoError(s.T(), err)
	s.sqlMock.ExpectationsWereMet() 
	// Potentially check for a specific log or a "skipped_no_associations" history record if that was implemented
}


func (s *ElasticScalingServiceTestSuite) TestEvaluateStrategy_NoSnapshots() {
	strategy := &portal.ElasticScalingStrategy{
		ID:              1,
		Name:            "TestNoSnapshots",
		Status:          StrategyStatusEnabled,
		CooldownMinutes: 60,
		DurationMinutes: 30,
		ResourceTypes:   "total",
		// ... other necessary fields
	}
	clusterID := int64(101)
	
	s.redisHandler.On("AcquireLock", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
	s.redisHandler.On("Delete", mock.Anything).Return()

	// Mock cooldown check (no recent history)
	s.sqlMock.ExpectQuery("SELECT \\* FROM `strategy_execution_histories`").
		WithArgs(strategy.ID, StrategyExecutionResultOrderCreated).
		WillReturnError(gorm.ErrRecordNotFound)

	// Mock associations (return one association)
	assocRows := sqlmock.NewRows([]string{"id", "strategy_id", "cluster_id"}).
		AddRow(1, strategy.ID, clusterID)
	s.sqlMock.ExpectQuery("SELECT \\* FROM `strategy_cluster_associations`").
		WithArgs(strategy.ID).
		WillReturnRows(assocRows)

	// Mock snapshot query for duration (return no rows)
	// For "total" resource type, resource_pool filter is not applied
	s.sqlMock.ExpectQuery("SELECT \\* FROM `resource_snapshots` WHERE cluster_id = \\? AND resource_type = \\? AND created_at BETWEEN \\? AND \\? ORDER BY created_at ASC").
		WithArgs(clusterID, "total", sqlmock.AnyArg(), sqlmock.AnyArg()). 
		WillReturnRows(sqlmock.NewRows([]string{"id"})) // No rows

	// Expect recordStrategyExecution call for no snapshots
	s.sqlMock.ExpectBegin()
	s.sqlMock.ExpectExec("INSERT INTO `strategy_execution_histories`").
		WithArgs(strategy.ID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), StrategyExecutionResultFailureNoSnapshots, "", "").
		WillReturnResult(sqlmock.NewResult(1, 1))
	s.sqlMock.ExpectCommit()
	
	err := s.service.evaluateStrategy(strategy)
	assert.NoError(s.T(), err)
	s.sqlMock.ExpectationsWereMet()
}


func (s *ElasticScalingServiceTestSuite) TestCheckConsistentThresholdBreach_CPUUsagePoolEntry_Breached() {
	strategy := &portal.ElasticScalingStrategy{
		ID:                     1,
		CPUThresholdValue:      80,
		CPUThresholdType:       ThresholdTypeUsage,
		ThresholdTriggerAction: TriggerActionPoolEntry,
		ConditionLogic:         ConditionLogicOr, // Doesn't matter if only one threshold
		DurationMinutes:        10,
	}
	snapshots := []portal.ResourceSnapshot{
		{MaxCpuUsageRatio: 85, CreatedAt: portal.NavyTime(time.Now().Add(-1 * time.Minute))},
		{MaxCpuUsageRatio: 90, CreatedAt: portal.NavyTime(time.Now().Add(-2 * time.Minute))},
		{MaxCpuUsageRatio: 82, CreatedAt: portal.NavyTime(time.Now().Add(-3 * time.Minute))},
	}

	breached, triggeredVal, thresholdVal := s.service.checkConsistentThresholdBreach(snapshots, strategy)
	
	assert.True(s.T(), breached)
	assert.Contains(s.T(), triggeredVal, "CPU usage: 85.67% (avg)") // (85+90+82)/3
	assert.Equal(s.T(), "CPU usage > 80% for 10 mins", thresholdVal)
}

func (s *ElasticScalingServiceTestSuite) TestCheckConsistentThresholdBreach_MemoryAllocatedPoolExit_Breached() {
	strategy := &portal.ElasticScalingStrategy{
		ID:                     1,
		MemoryThresholdValue:   30,
		MemoryThresholdType:    ThresholdTypeAllocated,
		ThresholdTriggerAction: TriggerActionPoolExit,
		ConditionLogic:         ConditionLogicOr,
		DurationMinutes:        15,
	}
	snapshots := []portal.ResourceSnapshot{
		{MemRequest: 10, MemoryCapacity: 100, CreatedAt: portal.NavyTime(time.Now().Add(-1 * time.Minute))}, // 10%
		{MemRequest: 20, MemoryCapacity: 100, CreatedAt: portal.NavyTime(time.Now().Add(-2 * time.Minute))}, // 20%
		{MemRequest: 15, MemoryCapacity: 100, CreatedAt: portal.NavyTime(time.Now().Add(-3 * time.Minute))}, // 15%
	}

	breached, triggeredVal, thresholdVal := s.service.checkConsistentThresholdBreach(snapshots, strategy)

	assert.True(s.T(), breached)
	assert.Contains(s.T(), triggeredVal, "Memory allocated: 15.00% (avg)") // (10+20+15)/3
	assert.Equal(s.T(), "Memory allocated < 30% for 15 mins", thresholdVal)
}

func (s *ElasticScalingServiceTestSuite) TestCheckConsistentThresholdBreach_CPUAndMemory_AND_Breached_PoolEntry() {
	strategy := &portal.ElasticScalingStrategy{
		ID:                     1,
		CPUThresholdValue:      70,
		CPUThresholdType:       ThresholdTypeUsage,
		MemoryThresholdValue:   60,
		MemoryThresholdType:    ThresholdTypeAllocated,
		ThresholdTriggerAction: TriggerActionPoolEntry,
		ConditionLogic:         ConditionLogicAnd,
		DurationMinutes:        5,
	}
	snapshots := []portal.ResourceSnapshot{
		{MaxCpuUsageRatio: 75, MemRequest: 65, MemoryCapacity: 100, CreatedAt: portal.NavyTime(time.Now().Add(-1 * time.Minute))}, // CPU: 75%, MemAlloc: 65%
		{MaxCpuUsageRatio: 80, MemRequest: 70, MemoryCapacity: 100, CreatedAt: portal.NavyTime(time.Now().Add(-2 * time.Minute))}, // CPU: 80%, MemAlloc: 70%
	}

	breached, triggeredVal, thresholdVal := s.service.checkConsistentThresholdBreach(snapshots, strategy)

	assert.True(s.T(), breached)
	assert.Contains(s.T(), triggeredVal, "CPU usage: 77.50% (avg)")
	assert.Contains(s.T(), triggeredVal, "Memory allocated: 67.50% (avg)")
	assert.Equal(s.T(), "CPU usage > 70% AND Memory allocated > 60% for 5 mins", thresholdVal)
}

func (s *ElasticScalingServiceTestSuite) TestCheckConsistentThresholdBreach_NotBreached_Intermittent() {
	strategy := &portal.ElasticScalingStrategy{
		ID:                     1,
		CPUThresholdValue:      80,
		CPUThresholdType:       ThresholdTypeUsage,
		ThresholdTriggerAction: TriggerActionPoolEntry,
		DurationMinutes:        10,
	}
	snapshots := []portal.ResourceSnapshot{
		{MaxCpuUsageRatio: 85},
		{MaxCpuUsageRatio: 75}, // This one does not meet criteria
		{MaxCpuUsageRatio: 90},
	}

	breached, _, _ := s.service.checkConsistentThresholdBreach(snapshots, strategy)
	assert.False(s.T(), breached)
}


func (s *ElasticScalingServiceTestSuite) TestCheckConsistentThresholdBreach_NotBreached_BelowThreshold_PoolEntry() {
	strategy := &portal.ElasticScalingStrategy{
		ID:                     1,
		CPUThresholdValue:      80,
		CPUThresholdType:       ThresholdTypeUsage,
		ThresholdTriggerAction: TriggerActionPoolEntry,
		DurationMinutes:        10,
	}
	snapshots := []portal.ResourceSnapshot{
		{MaxCpuUsageRatio: 70},
		{MaxCpuUsageRatio: 75},
		{MaxCpuUsageRatio: 60},
	}

	breached, triggeredVal, thresholdVal := s.service.checkConsistentThresholdBreach(snapshots, strategy)
	assert.False(s.T(), breached)
	assert.Contains(s.T(), triggeredVal, "CPU usage: 68.33% (avg)")
	assert.Equal(s.T(), "CPU usage > 80% for 10 mins", thresholdVal)
}

func (s *ElasticScalingServiceTestSuite) TestEvaluateStrategy_ThresholdNotMet() {
	strategy := &portal.ElasticScalingStrategy{
		ID:                     1,
		Name:                   "TestThresholdNotMet",
		Status:                 StrategyStatusEnabled,
		CooldownMinutes:        60,
		DurationMinutes:        10,
		ResourceTypes:          "total",
		CPUThresholdValue:      80,
		CPUThresholdType:       ThresholdTypeUsage,
		ThresholdTriggerAction: TriggerActionPoolEntry,
	}
	clusterID := int64(101)

	s.redisHandler.On("AcquireLock", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
	s.redisHandler.On("Delete", mock.Anything).Return()

	s.sqlMock.ExpectQuery("SELECT \\* FROM `strategy_execution_histories`").WillReturnError(gorm.ErrRecordNotFound) // Cooldown
	
	assocRows := sqlmock.NewRows([]string{"id", "strategy_id", "cluster_id"}).AddRow(1, strategy.ID, clusterID)
	s.sqlMock.ExpectQuery("SELECT \\* FROM `strategy_cluster_associations`").WillReturnRows(assocRows) // Associations

	snapshotRows := sqlmock.NewRows([]string{"id", "cluster_id", "resource_type", "max_cpu_usage_ratio", "created_at"}).
		AddRow(1, clusterID, "total", 70.0, time.Now().Add(-1*time.Minute)).
		AddRow(2, clusterID, "total", 75.0, time.Now().Add(-2*time.Minute))
	s.sqlMock.ExpectQuery("SELECT \\* FROM `resource_snapshots` WHERE cluster_id = \\? AND resource_type = \\? AND created_at BETWEEN \\? AND \\? ORDER BY created_at ASC").
		WillReturnRows(snapshotRows) // Snapshots

	s.sqlMock.ExpectBegin()
	s.sqlMock.ExpectExec("INSERT INTO `strategy_execution_histories`").
		WithArgs(strategy.ID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), StrategyExecutionResultFailureThresholdNotMet, "CPU usage: 72.50% (avg)", "CPU usage > 80% for 10 mins").
		WillReturnResult(sqlmock.NewResult(1, 1))
	s.sqlMock.ExpectCommit()

	err := s.service.evaluateStrategy(strategy)
	assert.NoError(s.T(), err)
	s.sqlMock.ExpectationsWereMet()
	// Assert that matchDevicesForStrategy was NOT called can be implicitly done by not mocking calls matchDevicesForStrategy would make
}

// Mock an AnyTime argument matcher for sqlmock
type AnyTime struct{}

// Match satisfies sqlmock.Argument interface
func (a AnyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

// Helper function to prepare a strategy for testing matchDevicesForStrategy
func (s *ElasticScalingServiceTestSuite) prepareStrategyForDeviceMatching(action string, templateID int64, deviceCount int) *portal.ElasticScalingStrategy {
	strategy := &portal.ElasticScalingStrategy{
		ID:                     1,
		Name:                   "DeviceMatchingTest",
		Status:                 StrategyStatusEnabled,
		CooldownMinutes:        60,
		DurationMinutes:        10,
		ResourceTypes:          "total",
		CPUThresholdValue:      80, // Assume threshold was met for these tests
		CPUThresholdType:       ThresholdTypeUsage,
		ThresholdTriggerAction: action,
		DeviceCount:            deviceCount,
	}
	if action == TriggerActionPoolEntry {
		strategy.EntryQueryTemplateID = templateID
	} else {
		strategy.ExitQueryTemplateID = templateID
	}
	return strategy
}


func (s *ElasticScalingServiceTestSuite) TestMatchDevices_InvalidTemplateID() {
	strategy := s.prepareStrategyForDeviceMatching(TriggerActionPoolEntry, 0, 1) // Template ID is 0

	s.sqlMock.ExpectBegin()
	s.sqlMock.ExpectExec("INSERT INTO `strategy_execution_histories`").
		WithArgs(strategy.ID, sqlmock.AnyArg(), sqlmock.AnyArg(), "Query template ID is not set for action type pool-in on strategy ID 1.", StrategyExecutionResultFailureInvalidTemplateID, "triggered", "threshold").
		WillReturnResult(sqlmock.NewResult(1, 1))
	s.sqlMock.ExpectCommit()
	
	err := s.service.matchDevicesForStrategy(strategy, 101, "total", nil, "triggered", "threshold")
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "Query template ID is not set")
	s.sqlMock.ExpectationsWereMet()
}


func (s *ElasticScalingServiceTestSuite) TestMatchDevices_TemplateNotFound() {
	templateID := int64(99)
	strategy := s.prepareStrategyForDeviceMatching(TriggerActionPoolEntry, templateID, 1)

	// Mock DB to return gorm.ErrRecordNotFound for QueryTemplate
	s.sqlMock.ExpectQuery("SELECT \\* FROM `query_templates` WHERE `query_templates`.`id` = \\? ORDER BY `query_templates`.`id` LIMIT 1").
		WithArgs(templateID).
		WillReturnError(gorm.ErrRecordNotFound)

	s.sqlMock.ExpectBegin()
	s.sqlMock.ExpectExec("INSERT INTO `strategy_execution_histories`").
		WithArgs(strategy.ID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), StrategyExecutionResultFailureTemplateNotFound, "triggered", "threshold").
		WillReturnResult(sqlmock.NewResult(1, 1))
	s.sqlMock.ExpectCommit()

	err := s.service.matchDevicesForStrategy(strategy, 101, "total", nil, "triggered", "threshold")
	assert.Error(s.T(), err)
	s.sqlMock.ExpectationsWereMet()
}


func (s *ElasticScalingServiceTestSuite) TestMatchDevices_QueryDevicesError() {
    templateID := int64(1)
    strategy := s.prepareStrategyForDeviceMatching(TriggerActionPoolEntry, templateID, 1)
    filterGroup := FilterGroup{ID: "group1", Blocks: []FilterBlock{{ID: "block1", Type: FilterTypeDevice, Key: "status", ConditionType: ConditionTypeEqual, Value: "available"}}}
    groupsJSON, _ := json.Marshal([]FilterGroup{filterGroup})

    queryTemplateRows := sqlmock.NewRows([]string{"id", "name", "groups"}).
        AddRow(templateID, "Test Template", string(groupsJSON))
    s.sqlMock.ExpectQuery("SELECT \\* FROM `query_templates`").WithArgs(templateID).WillReturnRows(queryTemplateRows)
    
    // Mock the device query to return an error.
    // This regex will match the complex query generated by DeviceQueryService.
    // Note: This is a simplified representation; the actual query is much more complex.
    s.sqlMock.ExpectQuery("SELECT device\\.\\*, CASE WHEN device\\.`group` != '' OR lf\\.id IS NOT NULL OR tf\\.id IS NOT NULL OR \\(\\(device\\.`group` = '' OR device\\.`group` IS NULL\\) AND \\(device\\.cluster = '' OR device\\.cluster IS NULL\\) AND da\\.name IS NOT NULL AND da\\.name != ''\\) THEN TRUE ELSE FALSE END AS is_special").
        WillReturnError(errors.New("DB error during device query"))


    s.sqlMock.ExpectBegin()
    s.sqlMock.ExpectExec("INSERT INTO `strategy_execution_histories`").
        WithArgs(strategy.ID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), StrategyExecutionResultFailureDeviceQuery, "triggered", "threshold").
        WillReturnResult(sqlmock.NewResult(1, 1))
    s.sqlMock.ExpectCommit()

    err := s.service.matchDevicesForStrategy(strategy, 101, "total", nil, "triggered", "threshold")
    assert.Error(s.T(), err)
    s.sqlMock.ExpectationsWereMet()
}


func (s *ElasticScalingServiceTestSuite) TestMatchDevices_NoCandidateDevicesFound() {
    templateID := int64(1)
    strategy := s.prepareStrategyForDeviceMatching(TriggerActionPoolEntry, templateID, 1)
    filterGroup := FilterGroup{ID: "group1", Blocks: []FilterBlock{{ID: "block1", Type: FilterTypeDevice, Key: "status", ConditionType: ConditionTypeEqual, Value: "available"}}}
    groupsJSON, _ := json.Marshal([]FilterGroup{filterGroup})

    queryTemplateRows := sqlmock.NewRows([]string{"id", "name", "groups"}).
        AddRow(templateID, "Test Template", string(groupsJSON))
    s.sqlMock.ExpectQuery("SELECT \\* FROM `query_templates`").WithArgs(templateID).WillReturnRows(queryTemplateRows)
    
    // Mock the device query to return no rows
    s.sqlMock.ExpectQuery("SELECT device\\.\\*, CASE WHEN device\\.`group` != '' OR lf\\.id IS NOT NULL OR tf\\.id IS NOT NULL OR \\(\\(device\\.`group` = '' OR device\\.`group` IS NULL\\) AND \\(device\\.cluster = '' OR device\\.cluster IS NULL\\) AND da\\.name IS NOT NULL AND da\\.name != ''\\) THEN TRUE ELSE FALSE END AS is_special").
		WillReturnRows(sqlmock.NewRows([]string{"id"})) // No devices

    s.sqlMock.ExpectBegin()
    s.sqlMock.ExpectExec("INSERT INTO `strategy_execution_histories`").
        WithArgs(strategy.ID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), StrategyExecutionResultFailureNoDevicesFound, "triggered", "threshold").
        WillReturnResult(sqlmock.NewResult(1, 1))
    s.sqlMock.ExpectCommit()

    err := s.service.matchDevicesForStrategy(strategy, 101, "total", nil, "triggered", "threshold")
    assert.NoError(s.T(), err) // No error, but no devices found, so history recorded.
    s.sqlMock.ExpectationsWereMet()
}


// AnyInt64Value is a helper for sqlmock arguments for any int64
type AnyInt64Value struct{}
func (a AnyInt64Value) Match(v driver.Value) bool {
    _, ok := v.(int64)
    return ok
}


func (s *ElasticScalingServiceTestSuite) TestGenerateOrder_Success() {
	strategy := &portal.ElasticScalingStrategy{ID: 1, ThresholdTriggerAction: TriggerActionPoolEntry}
	clusterID := int64(101)
	selectedDeviceIDs := []int64{1, 2}
	triggeredValueStr := "CPU > 80%"
	thresholdValueStr := "CPU Usage: 85%" // Corrected from previous version

	s.sqlMock.ExpectBegin()
	// Mock ElasticScalingOrder creation
	s.sqlMock.ExpectExec("INSERT INTO `elastic_scaling_orders`").
		WithArgs(sqlmock.AnyArg(), clusterID, strategy.ID, strategy.ThresholdTriggerAction, OrderStatusPending, len(selectedDeviceIDs), nil, SystemAutoCreator, sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, "", triggeredValueStr, thresholdValueStr). 
		WillReturnResult(sqlmock.NewResult(1, 1)) // New order ID 1
	s.sqlMock.ExpectCommit()
	
	s.sqlMock.ExpectBegin()
	// Mock OrderDevice creation for device 1
	s.sqlMock.ExpectExec("INSERT INTO `order_devices`").
		WithArgs(AnyInt64Value{}, int64(1), int64(1), OrderStatusPending). // OrderID 1, DeviceID 1
		WillReturnResult(sqlmock.NewResult(1,1))
	s.sqlMock.ExpectCommit()

	s.sqlMock.ExpectBegin()
	// Mock OrderDevice creation for device 2
	s.sqlMock.ExpectExec("INSERT INTO `order_devices`").
		WithArgs(AnyInt64Value{}, int64(1), int64(2), OrderStatusPending). // OrderID 1, DeviceID 2
		WillReturnResult(sqlmock.NewResult(2,1))
	s.sqlMock.ExpectCommit()


	s.sqlMock.ExpectBegin()
	// Mock strategy execution history recording
	s.sqlMock.ExpectExec("INSERT INTO `strategy_execution_histories`").
		WithArgs(strategy.ID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), StrategyExecutionResultOrderCreated, AnyInt64Value{}, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	s.sqlMock.ExpectCommit()

	err := s.service.generateElasticScalingOrder(strategy, clusterID, "total", selectedDeviceIDs, triggeredValueStr, thresholdValueStr)
	assert.NoError(s.T(), err)
	s.sqlMock.ExpectationsWereMet()
}


func (s *ElasticScalingServiceTestSuite) TestGenerateOrder_CreateOrderFails() {
    strategy := &portal.ElasticScalingStrategy{ID: 1, ThresholdTriggerAction: TriggerActionPoolEntry}
    clusterID := int64(101)
    selectedDeviceIDs := []int64{1, 2}

    s.sqlMock.ExpectBegin()
    s.sqlMock.ExpectExec("INSERT INTO `elastic_scaling_orders`").
        WillReturnError(errors.New("DB order creation failed"))
    s.sqlMock.ExpectRollback() 


    s.sqlMock.ExpectBegin()
    s.sqlMock.ExpectExec("INSERT INTO `strategy_execution_histories`").
        WithArgs(strategy.ID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), StrategyExecutionResultOrderFailed, nil, sqlmock.AnyArg()).
        WillReturnResult(sqlmock.NewResult(1, 1))
    s.sqlMock.ExpectCommit()
    
    err := s.service.generateElasticScalingOrder(strategy, clusterID, "total", selectedDeviceIDs, "triggered", "threshold")
    assert.Error(s.T(), err)
    assert.Contains(s.T(), err.Error(), "DB order creation failed")
    s.sqlMock.ExpectationsWereMet()
}


func (s *ElasticScalingServiceTestSuite) TestGenerateOrder_CreateOrderDeviceFails() {
    strategy := &portal.ElasticScalingStrategy{ID: 1, ThresholdTriggerAction: TriggerActionPoolEntry}
    clusterID := int64(101)
    selectedDeviceIDs := []int64{1, 2}
    triggeredValueStr := "CPU > 80%"
    thresholdValueStr := "CPU Usage: 85%"

    s.sqlMock.ExpectBegin()
    // Mock ElasticScalingOrder creation - success
    s.sqlMock.ExpectExec("INSERT INTO `elastic_scaling_orders`").
        WithArgs(sqlmock.AnyArg(), clusterID, strategy.ID, strategy.ThresholdTriggerAction, OrderStatusPending, len(selectedDeviceIDs), nil, SystemAutoCreator, sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, "", triggeredValueStr, thresholdValueStr).
        WillReturnResult(sqlmock.NewResult(1, 1)) // New order ID 1
    s.sqlMock.ExpectCommit()

    s.sqlMock.ExpectBegin()
    // Mock OrderDevice creation for device 1 - fail
    s.sqlMock.ExpectExec("INSERT INTO `order_devices`").
        WithArgs(AnyInt64Value{}, int64(1), int64(1), OrderStatusPending).
        WillReturnError(errors.New("DB order device creation failed"))
    s.sqlMock.ExpectRollback() // GORM will rollback this transaction

    // No history should be recorded by generateElasticScalingOrder in this specific partial failure,
    // as CreateOrder itself returns an error. generateElasticScalingOrder will then record OrderFailed.
    s.sqlMock.ExpectBegin()
    s.sqlMock.ExpectExec("INSERT INTO `strategy_execution_histories`").
        WithArgs(strategy.ID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), StrategyExecutionResultOrderFailed, nil, sqlmock.AnyArg()).
        WillReturnResult(sqlmock.NewResult(1, 1))
    s.sqlMock.ExpectCommit()

    err := s.service.generateElasticScalingOrder(strategy, clusterID, "total", selectedDeviceIDs, triggeredValueStr, thresholdValueStr)
    assert.Error(s.T(), err)
    assert.Contains(s.T(), err.Error(), "订单创建成功，但关联设备时出错") // This is the error from CreateOrder
    s.sqlMock.ExpectationsWereMet()
}


func (s *ElasticScalingServiceTestSuite) TestMatchDevices_PoolEntry_SelectsAvailableDevices() {
    templateID := int64(1)
    strategy := s.prepareStrategyForDeviceMatching(TriggerActionPoolEntry, templateID, 2) // Expect 2 devices
    currentClusterID := int64(101) // The cluster triggering the strategy, not where devices are necessarily from for pool-in

    filterGroup := FilterGroup{ID: "group1", Blocks: []FilterBlock{{ID: "block1", Type: FilterTypeDevice, Key: "status", ConditionType: ConditionTypeEqual, Value: "available"}}}
    groupsJSON, _ := json.Marshal([]FilterGroup{filterGroup})
    queryTemplateRows := sqlmock.NewRows([]string{"id", "name", "groups"}).AddRow(templateID, "Avail Template", string(groupsJSON))
    s.sqlMock.ExpectQuery("SELECT \\* FROM `query_templates`").WithArgs(templateID).WillReturnRows(queryTemplateRows)

    // Mock device query result
    deviceRows := sqlmock.NewRows([]string{"id", "ci_code", "cluster_id", "cluster"}).
        AddRow(1, "dev1", int64(0), "").         // Available
        AddRow(2, "dev2", int64(102), "other"). // In another cluster
        AddRow(3, "dev3", int64(0), "")          // Available
    s.sqlMock.ExpectQuery("SELECT device\\.\\*").WillReturnRows(deviceRows) // Simplified regex for device query

    // Expect order generation with device IDs 1 and 3
    s.sqlMock.ExpectBegin() 
    s.sqlMock.ExpectExec("INSERT INTO `elastic_scaling_orders`").WithArgs(sqlmock.AnyArg(), currentClusterID, strategy.ID, strategy.ThresholdTriggerAction, OrderStatusPending, 2, nil, SystemAutoCreator, sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, "", "triggered", "threshold").WillReturnResult(sqlmock.NewResult(1, 1))
    s.sqlMock.ExpectCommit()

    s.sqlMock.ExpectBegin() 
    s.sqlMock.ExpectExec("INSERT INTO `order_devices`").WithArgs(AnyInt64Value{}, int64(1), int64(1), OrderStatusPending).WillReturnResult(sqlmock.NewResult(1,1))
    s.sqlMock.ExpectCommit()
    s.sqlMock.ExpectBegin() 
    s.sqlMock.ExpectExec("INSERT INTO `order_devices`").WithArgs(AnyInt64Value{}, int64(1), int64(3), OrderStatusPending).WillReturnResult(sqlmock.NewResult(2,1))
    s.sqlMock.ExpectCommit()

    s.sqlMock.ExpectBegin() 
    s.sqlMock.ExpectExec("INSERT INTO `strategy_execution_histories`").WithArgs(strategy.ID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), StrategyExecutionResultOrderCreated, AnyInt64Value{}, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
    s.sqlMock.ExpectCommit()

    err := s.service.matchDevicesForStrategy(strategy, currentClusterID, "total", nil, "triggered", "threshold")
    assert.NoError(s.T(), err)
    s.sqlMock.ExpectationsWereMet()
}


func (s *ElasticScalingServiceTestSuite) TestMatchDevices_PoolExit_SelectsClusterDevices() {
    templateID := int64(2)
    targetClusterID := int64(10)
    strategy := s.prepareStrategyForDeviceMatching(TriggerActionPoolExit, templateID, 1) // Expect 1 device
    
    filterGroup := FilterGroup{ID: "group1", Blocks: []FilterBlock{{ID: "block1", Type: FilterTypeDevice, Key: "status", ConditionType: ConditionTypeEqual, Value: "in-use"}}}
    groupsJSON, _ := json.Marshal([]FilterGroup{filterGroup})
    queryTemplateRows := sqlmock.NewRows([]string{"id", "name", "groups"}).AddRow(templateID, "Exit Template", string(groupsJSON))
    s.sqlMock.ExpectQuery("SELECT \\* FROM `query_templates`").WithArgs(templateID).WillReturnRows(queryTemplateRows)

    deviceRows := sqlmock.NewRows([]string{"id", "ci_code", "cluster_id", "cluster"}).
        AddRow(1, "dev1", targetClusterID, "cluster10"). // In target cluster
        AddRow(2, "dev2", int64(20), "cluster20").      // In another cluster
        AddRow(3, "dev3", int64(0), "")                 // Available (not in target cluster)
    s.sqlMock.ExpectQuery("SELECT device\\.\\*").WillReturnRows(deviceRows)

    s.sqlMock.ExpectBegin()
    s.sqlMock.ExpectExec("INSERT INTO `elastic_scaling_orders`").WithArgs(sqlmock.AnyArg(), targetClusterID, strategy.ID, strategy.ThresholdTriggerAction, OrderStatusPending, 1, nil, SystemAutoCreator, sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, "", "triggered", "threshold").WillReturnResult(sqlmock.NewResult(1, 1))
    s.sqlMock.ExpectCommit()

    s.sqlMock.ExpectBegin()
    s.sqlMock.ExpectExec("INSERT INTO `order_devices`").WithArgs(AnyInt64Value{}, int64(1), int64(1), OrderStatusPending).WillReturnResult(sqlmock.NewResult(1,1))
    s.sqlMock.ExpectCommit()

    s.sqlMock.ExpectBegin()
    s.sqlMock.ExpectExec("INSERT INTO `strategy_execution_histories`").WithArgs(strategy.ID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), StrategyExecutionResultOrderCreated, AnyInt64Value{}, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
    s.sqlMock.ExpectCommit()

    err := s.service.matchDevicesForStrategy(strategy, targetClusterID, "total", nil, "triggered", "threshold")
    assert.NoError(s.T(), err)
    s.sqlMock.ExpectationsWereMet()
}

func (s *ElasticScalingServiceTestSuite) TestMatchDevices_PoolExit_NoDevicesInTargetCluster() {
    templateID := int64(3)
    targetClusterID := int64(10)
    strategy := s.prepareStrategyForDeviceMatching(TriggerActionPoolExit, templateID, 1)
    
    filterGroup := FilterGroup{ID: "group1", Blocks: []FilterBlock{{ID: "block1", Type: FilterTypeDevice, Key: "status", ConditionType: ConditionTypeEqual, Value: "in-use"}}}
    groupsJSON, _ := json.Marshal([]FilterGroup{filterGroup})
    queryTemplateRows := sqlmock.NewRows([]string{"id", "name", "groups"}).AddRow(templateID, "Exit Template", string(groupsJSON))
    s.sqlMock.ExpectQuery("SELECT \\* FROM `query_templates`").WithArgs(templateID).WillReturnRows(queryTemplateRows)

    deviceRows := sqlmock.NewRows([]string{"id", "ci_code", "cluster_id", "cluster"}).
        AddRow(1, "dev1", int64(20), "cluster20"). // In another cluster
        AddRow(2, "dev2", int64(0), "")            // Available
    s.sqlMock.ExpectQuery("SELECT device\\.\\*").WillReturnRows(deviceRows)

    s.sqlMock.ExpectBegin()
    s.sqlMock.ExpectExec("INSERT INTO `strategy_execution_histories`").
        WithArgs(strategy.ID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), StrategyExecutionResultFailureNoSuitableDevices, "triggered", "threshold").
        WillReturnResult(sqlmock.NewResult(1, 1))
    s.sqlMock.ExpectCommit()

    err := s.service.matchDevicesForStrategy(strategy, targetClusterID, "total", nil, "triggered", "threshold")
    assert.NoError(s.T(), err) // No error, but history recorded for no suitable devices
    s.sqlMock.ExpectationsWereMet()
}


func (s *ElasticScalingServiceTestSuite) TestMatchDevices_DeviceCountExceedsCandidates() {
    templateID := int64(4)
    strategy := s.prepareStrategyForDeviceMatching(TriggerActionPoolEntry, templateID, 3) // Expect 3 devices
    currentClusterID := int64(101)

    filterGroup := FilterGroup{ID: "group1", Blocks: []FilterBlock{{ID: "block1", Type: FilterTypeDevice, Key: "status", ConditionType: ConditionTypeEqual, Value: "available"}}}
    groupsJSON, _ := json.Marshal([]FilterGroup{filterGroup})
    queryTemplateRows := sqlmock.NewRows([]string{"id", "name", "groups"}).AddRow(templateID, "Avail Template", string(groupsJSON))
    s.sqlMock.ExpectQuery("SELECT \\* FROM `query_templates`").WithArgs(templateID).WillReturnRows(queryTemplateRows)

    // Only 2 devices are returned by the query
    deviceRows := sqlmock.NewRows([]string{"id", "ci_code", "cluster_id", "cluster"}).
        AddRow(1, "dev1", int64(0), "").
        AddRow(2, "dev2", int64(0), "")
    s.sqlMock.ExpectQuery("SELECT device\\.\\*").WillReturnRows(deviceRows)

    // Expect order generation with 2 device IDs (all available)
    s.sqlMock.ExpectBegin()
    s.sqlMock.ExpectExec("INSERT INTO `elastic_scaling_orders`").WithArgs(sqlmock.AnyArg(), currentClusterID, strategy.ID, strategy.ThresholdTriggerAction, OrderStatusPending, 2, nil, SystemAutoCreator, sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, "", "triggered", "threshold").WillReturnResult(sqlmock.NewResult(1, 1))
    s.sqlMock.ExpectCommit()

    s.sqlMock.ExpectBegin()
    s.sqlMock.ExpectExec("INSERT INTO `order_devices`").WithArgs(AnyInt64Value{}, int64(1), int64(1), OrderStatusPending).WillReturnResult(sqlmock.NewResult(1,1))
    s.sqlMock.ExpectCommit()
    s.sqlMock.ExpectBegin()
    s.sqlMock.ExpectExec("INSERT INTO `order_devices`").WithArgs(AnyInt64Value{}, int64(1), int64(2), OrderStatusPending).WillReturnResult(sqlmock.NewResult(2,1))
    s.sqlMock.ExpectCommit()

    s.sqlMock.ExpectBegin()
    s.sqlMock.ExpectExec("INSERT INTO `strategy_execution_histories`").WithArgs(strategy.ID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), StrategyExecutionResultOrderCreated, AnyInt64Value{}, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
    s.sqlMock.ExpectCommit()

    err := s.service.matchDevicesForStrategy(strategy, currentClusterID, "total", nil, "triggered", "threshold")
    assert.NoError(s.T(), err)
    s.sqlMock.ExpectationsWereMet()
}


func (s *ElasticScalingServiceTestSuite) TestEvaluateStrategy_ThresholdMet_CallsMatchDevicesAndOrder() {
    strategy := &portal.ElasticScalingStrategy{
        ID:                     1,
        Name:                   "TestThresholdMetOrder",
        Status:                 StrategyStatusEnabled,
        CooldownMinutes:        60,
        DurationMinutes:        10,
        ResourceTypes:          "total",
        CPUThresholdValue:      80,
        CPUThresholdType:       ThresholdTypeUsage,
        ThresholdTriggerAction: TriggerActionPoolEntry,
        EntryQueryTemplateID:   int64(123), // Valid template ID
        DeviceCount:            1,
    }
    clusterID := int64(101)
    templateID := strategy.EntryQueryTemplateID

    s.redisHandler.On("AcquireLock", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
    s.redisHandler.On("Delete", mock.Anything).Return()

    s.sqlMock.ExpectQuery("SELECT \\* FROM `strategy_execution_histories` WHERE strategy_id = \\? AND result = \\? ORDER BY execution_time DESC LIMIT 1").WithArgs(strategy.ID, StrategyExecutionResultOrderCreated).WillReturnError(gorm.ErrRecordNotFound) // Cooldown

    assocRows := sqlmock.NewRows([]string{"id", "strategy_id", "cluster_id"}).AddRow(1, strategy.ID, clusterID)
    s.sqlMock.ExpectQuery("SELECT \\* FROM `strategy_cluster_associations`").WithArgs(strategy.ID).WillReturnRows(assocRows) // Associations

    // Snapshots that will cause a breach
    snapshotRows := sqlmock.NewRows([]string{"id", "cluster_id", "resource_type", "max_cpu_usage_ratio", "created_at"}).
        AddRow(1, clusterID, "total", 85.0, time.Now().Add(-1*time.Minute)).
        AddRow(2, clusterID, "total", 90.0, time.Now().Add(-2*time.Minute))
    s.sqlMock.ExpectQuery("SELECT \\* FROM `resource_snapshots` WHERE cluster_id = \\? AND resource_type = \\? AND created_at BETWEEN \\? AND \\? ORDER BY created_at ASC").
        WillReturnRows(snapshotRows)

    // --- Mocks for matchDevicesForStrategy ---
    filterGroup := FilterGroup{ID: "group1", Blocks: []FilterBlock{{ID: "block1", Type: FilterTypeDevice, Key: "status", ConditionType: ConditionTypeEqual, Value: "available"}}}
    groupsJSON, _ := json.Marshal([]FilterGroup{filterGroup})
    queryTemplateRows := sqlmock.NewRows([]string{"id", "name", "groups"}).AddRow(templateID, "Test Template", string(groupsJSON))
    s.sqlMock.ExpectQuery("SELECT \\* FROM `query_templates` WHERE `query_templates`.`id` = \\? ORDER BY `query_templates`.`id` LIMIT 1").WithArgs(templateID).WillReturnRows(queryTemplateRows)

    deviceRows := sqlmock.NewRows([]string{"id", "ci_code", "cluster_id", "cluster"}).AddRow(int64(55), "dev55", int64(0), "")
    s.sqlMock.ExpectQuery("SELECT device\\.\\*").WillReturnRows(deviceRows) // Device query

    // --- Mocks for generateElasticScalingOrder ---
    s.sqlMock.ExpectBegin() // For CreateOrder
    s.sqlMock.ExpectExec("INSERT INTO `elastic_scaling_orders`").WithArgs(sqlmock.AnyArg(), clusterID, strategy.ID, strategy.ThresholdTriggerAction, OrderStatusPending, strategy.DeviceCount, nil, SystemAutoCreator, sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, "", "CPU usage: 87.50% (avg)", "CPU usage > 80% for 10 mins").WillReturnResult(sqlmock.NewResult(1, 1))
    s.sqlMock.ExpectCommit()

    s.sqlMock.ExpectBegin() // For OrderDevice
    s.sqlMock.ExpectExec("INSERT INTO `order_devices`").WithArgs(AnyInt64Value{}, int64(1), int64(55), OrderStatusPending).WillReturnResult(sqlmock.NewResult(1,1))
    s.sqlMock.ExpectCommit()

    s.sqlMock.ExpectBegin() // For history record (order_created)
    s.sqlMock.ExpectExec("INSERT INTO `strategy_execution_histories`").WithArgs(strategy.ID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), StrategyExecutionResultOrderCreated, AnyInt64Value{}, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
    s.sqlMock.ExpectCommit()
    
    err := s.service.evaluateStrategy(strategy)
    assert.NoError(s.T(), err)
    s.sqlMock.ExpectationsWereMet()
}
