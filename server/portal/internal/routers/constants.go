package routers

// HTTP 路由路径常量
const (
	// 基础路由组
	RouteGroupDevices              = "/device"
	RouteGroupDeviceQuery          = "/device-query"
	RouteGroupK8sClusters          = "/k8s-clusters"
	RouteGroupResourcePool         = "/resource-pool"
	RouteGroupElasticScaling       = "/elastic-scaling"
	RouteGroupElasticScalingOrders = "/elastic-scaling/orders"
	RouteGroupClusterResources     = "/cluster-resources"
	RouteGroupDeviceMaintenance    = "/fe-v1/device-maintenance"

	// 路由参数路径
	RouteParamID                = "/:id"
	RouteParamIDStatus          = "/:id/status"
	RouteParamIDDevices         = "/:id/devices"
	RouteParamIDDevicesDeviceID = "/:id/devices/:device_id"
	RouteParamIDNodes           = "/:id/nodes"
	RouteParamIDGroup           = "/:id/group"

	// 子路由路径
	SubRouteExport               = "/export"
	SubRouteFilterOptions        = "/filter-options"
	SubRouteLabelValues          = "/label-values"
	SubRouteTaintValues          = "/taint-values"
	SubRouteDeviceFieldValues    = "/device-field-values"
	SubRouteDeviceFeatureDetails = "/device-feature-details"
	SubRouteQuery                = "/query"
	SubRouteTemplates            = "/templates"
	SubRouteByType               = "/by-type"
	SubRouteMatchingPolicies     = "/matching-policies"
	SubRouteRemaining            = "/remaining"
	SubRouteAllocationRate       = "/allocation-rate"
	SubRouteRequest              = "/request"
)

// HTTP 参数名常量
const (
	ParamID               = "id"
	ParamOrderType        = "orderType"
	ParamDeviceID         = "device_id"
	ParamTimeRange        = "timeRange"
	ParamPage             = "page"
	ParamSize             = "size"
	ParamName             = "name"
	ParamStatus           = "status"
	ParamCICode           = "ci_code"
	ParamDate             = "date"
	ParamPurposeFilter    = "purpose_filter"
	ParamResourcePoolType = "resourcePoolType"
	ParamActionType       = "actionType"
)

// HTTP 查询参数默认值
const (
	DefaultPageValue = "1"
	DefaultSizeValue = "10"
	DefaultPageInt   = 1
	DefaultSizeInt   = 10
	MaxSizeInt       = 100
)

// 数据库和缓存相关常量
const (
	RedisDefault   = "default"
	RedisNamespace = "navy"
	RedisVersion   = "v1"
	Base10         = 10
	BitSize64      = 64
)

// 订单类型常量
const (
	OrderTypeGeneral = "general"
)

// 状态常量
const (
	StatusEnabled   = "enabled"
	StatusDisabled  = "disabled"
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

// 用户相关常量
const (
	DefaultExecutor    = "admin"
	DefaultUsername    = "system"
	UsernameContextKey = "username"
	BasicAuthUser      = "admin"
	BasicAuthPassword  = "password"
	BasicAuthRealm     = `Basic realm="Restricted"`
)

// HTTP 响应消息常量
const (
	MsgSuccess        = "success"
	MsgUnauthorized   = "Unauthorized"
	MsgRequestTimeout = "请求超时，请稍后重试"
)

// 通用错误消息常量
const (
	// ID 相关错误
	MsgInvalidID             = "无效的ID"
	MsgInvalidIDFormat       = "invalid id format"
	MsgInvalidDeviceIDFormat = "invalid device id format"
	MsgInvalidJobIDFormat    = "invalid job id format"
	MsgInvalidStrategyID     = "无效的策略ID"
	MsgInvalidOrderID        = "无效的订单ID"
	MsgInvalidPolicyID       = "invalid policy ID"

	// 参数相关错误
	MsgInvalidParams        = "参数解析失败: "
	MsgInvalidQueryParams   = "无效的查询参数: "
	MsgInvalidRequestParams = "无效的请求参数: "
	MsgInvalidRequestBody   = "invalid request body: %s"
	MsgInvalidRequestFormat = "无效的请求格式: "
	MsgParamBindFailed      = "参数绑定失败: %s"
	MsgURIBindFailed        = "URI参数绑定失败: %s"
	MsgBodyBindFailed       = "请求体绑定失败: %s"

	// 业务操作错误
	MsgFailedToList   = "failed to list: %s"
	MsgFailedToGet    = "failed to get: %s"
	MsgFailedToCreate = "failed to create: %s"
	MsgFailedToUpdate = "failed to update: %s"
	MsgFailedToDelete = "failed to delete: %s"
	MsgFailedToExport = "failed to export: %s"

	// 设备相关错误
	MsgFailedToListDevices       = "failed to list devices: %s"
	MsgFailedToGetDevice         = "failed to get device: %s"
	MsgFailedToQueryDevices      = "failed to query devices: %s"
	MsgFailedToExportDevices     = "failed to export devices: %s"
	MsgFailedToUpdateDeviceRole  = "failed to update device role: %s"
	MsgFailedToUpdateDeviceGroup = "failed to update device group: %s"
	MsgInvalidRoleUpdateRequest  = "invalid role update request: %s"
	MsgInvalidGroupUpdateRequest = "invalid group update request: %s"

	// 集群相关错误
	MsgFailedToGetClusters     = "获取集群列表失败: "
	MsgFailedToGetCluster      = "获取集群失败: "
	MsgFailedToCreateCluster   = "创建集群失败: "
	MsgFailedToUpdateCluster   = "更新集群失败: "
	MsgFailedToDeleteCluster   = "删除集群失败: "
	MsgFailedToGetClusterNodes = "获取集群节点失败: "

	// 订单相关错误
	MsgInvalidOrderType     = "无效或未注册的订单类型: %s"
	MsgUnsupportedOrderType = "不支持的订单类型: %s"
	MsgServiceTypeMismatch  = "服务接口类型不匹配"
	MsgGetOrderFailed       = "获取订单失败: %s"
	MsgCreateOrderFailed    = "创建订单失败: %s"
	MsgListOrdersFailed     = "获取订单列表失败: %s"
	MsgUpdateStatusFailed   = "更新订单状态失败: %s"

	// 作业相关错误
	MsgFailedToListJobs      = "failed to list operation jobs: %s"
	MsgFailedToCreateJob     = "failed to create operation job: %s"
	MsgInvalidJobQueryParams = "invalid query parameters: %s"
	MsgInvalidJobBody        = "invalid request body: %s"

	// F5 相关错误
	MsgFailedToListF5   = "failed to list F5 infos: %s"
	MsgFailedToUpdateF5 = "failed to update F5 info: %s"
	MsgFailedToDeleteF5 = "failed to delete F5 info: %s"

	// 查询相关错误
	MsgFailedToGetFilterOptions     = "failed to get filter options: %s"
	MsgFailedToGetLabelValues       = "failed to get label values: %s"
	MsgFailedToGetTaintValues       = "failed to get taint values: %s"
	MsgFailedToGetDeviceFieldValues = "failed to get device field values: %s"
	MsgInvalidQueryRequest          = "invalid query request: %s"

	// 模板相关错误
	MsgFailedToSaveTemplate   = "failed to save template: %s"
	MsgFailedToGetTemplates   = "failed to get templates: %s"
	MsgFailedToGetTemplate    = "failed to get template: %s"
	MsgFailedToDeleteTemplate = "failed to delete template: %s"

	// 策略相关错误
	MsgFailedToGetPolicies = "获取匹配策略失败: "
	MsgInvalidStatus       = "状态必须为 enabled 或 disabled"

	// WebSocket 相关错误
	MsgWebSocketUpgradeError = "failed to upgrade to websocket: %s"

	// 维护相关错误
	MsgInvalidMaintenanceRequest = "无效的请求格式: "
	MsgDeviceIDOrCICodeRequired  = "设备ID或CI编码不能为空"
)

// 成功消息常量
const (
	// 通用成功消息
	MsgOperationSuccess = "操作成功"
	MsgCreatedSuccess   = "创建成功"
	MsgUpdatedSuccess   = "更新成功"
	MsgDeletedSuccess   = "删除成功"

	// 设备相关成功消息
	MsgDeviceRoleUpdated  = "device role updated successfully"
	MsgDeviceGroupUpdated = "device group updated successfully"

	// F5 相关成功消息
	MsgF5UpdateSuccess = "F5 info updated successfully"
	MsgF5DeleteSuccess = "F5 info deleted successfully"

	// 作业相关成功消息
	MsgJobCreatedSuccess = "Operation job created successfully"

	// 策略相关成功消息
	MsgPolicyUpdatedSuccess       = "policy updated successfully"
	MsgPolicyDeletedSuccess       = "policy deleted successfully"
	MsgPolicyStatusUpdatedSuccess = "policy status updated successfully"
)

// HTTP 头部和内容类型常量
const (
	HeaderContentDescription      = "Content-Description"
	HeaderContentDisposition      = "Content-Disposition"
	HeaderContentType             = "Content-Type"
	HeaderContentTransferEncoding = "Content-Transfer-Encoding"
	HeaderExpires                 = "Expires"
	HeaderCacheControl            = "Cache-Control"
	HeaderPragma                  = "Pragma"
	HeaderContentLength           = "Content-Length"
	HeaderWWWAuthenticate         = "WWW-Authenticate"

	ContentTypeCSV  = "text/csv"
	ContentTypeJSON = "application/json"
)

// 时间格式常量
const (
	DateFormatYYYYMMDD = "2006-01-02"
)
