package routers

import (
	"navy-ng/pkg/middleware/render"
	"navy-ng/server/portal/internal/service"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ClusterResourceHandler handles API requests related to cluster resources.
type ClusterResourceHandler struct {
	service *service.ClusterResourceService
}

// NewClusterResourceHandler creates a new ClusterResourceHandler instance.
func NewClusterResourceHandler(db *gorm.DB) *ClusterResourceHandler {
	return &ClusterResourceHandler{
		service: service.NewClusterResourceService(db),
	}
}

// RegisterRoutes registers the cluster resource-related routes.
func (h *ClusterResourceHandler) RegisterRoutes(api *gin.RouterGroup) {
	clusterResourcesGroup := api.Group(RouteGroupClusterResources)
	{
		clusterResourcesGroup.GET(SubRouteRemaining, h.GetRemainingClusterResources)
		clusterResourcesGroup.GET(SubRouteAllocationRate, h.GetResourcePoolAllocationRate)
	}
}

// GetRemainingClusterResources retrieves the calculated remaining cluster resources.
// @Summary Get remaining cluster resources
// @Description Retrieves the calculated remaining cluster resources based on snapshots for specified date (defaults to today) with optional description filter.
// @Tags Cluster Resources
// @Accept json
// @Produce json
// @Param date query string false "Query date in YYYY-MM-DD format (defaults to today)"
// @Param purpose_filter query string false "Regex pattern to filter clusters by description field"
// @Success 200 {object} service.ClusterResourceDTO "Successfully retrieved remaining resources"
// @Failure 400 {object} render.ErrorResponse "Invalid date format"
// @Failure 500 {object} render.ErrorResponse "Internal server error"
// @Router /api/cluster-resources/remaining [get]
type ClusterResourceQuery struct {
	Date          string `form:"date"`
	PurposeFilter string `form:"purpose_filter"`
}

func (h *ClusterResourceHandler) GetRemainingClusterResources(c *gin.Context) {
	var query ClusterResourceQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		render.BadRequest(c, MsgInvalidParams+err.Error())
		return
	}

	// Parse optional date parameter
	var queryDate *time.Time
	if query.Date != "" {
		parsedDate, err := time.Parse(time.DateOnly, query.Date)
		if err != nil {
			render.Fail(c, http.StatusBadRequest, "Invalid date format. Please use YYYY-MM-DD format.")
			return
		}
		queryDate = &parsedDate
	}

	// Parse optional description filter parameter
	var descFilter *string
	if query.PurposeFilter != "" {
		descFilter = &query.PurposeFilter
	}

	// Calculate remaining resources with optional date and description filter
	resources, err := h.service.CalculateRemainingResources(queryDate, descFilter)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, "Failed to calculate remaining cluster resources: "+err.Error())
		return
	}

	// The service.ClusterResourceDTO already has Code, Message, and List fields.
	// If the service returns a DTO with Code=0 and Message="No resource snapshots found for specified date",
	// it will be rendered as such, which is acceptable.
	// If there's a genuine calculation success, it will also be structured correctly.
	c.JSON(http.StatusOK, resources)
}

// GetResourcePoolAllocationRate retrieves CPU and memory allocation rates for a specific cluster and resource pool.
// @Summary Get resource pool allocation rate
// @Description Retrieves CPU and memory allocation rates for a specific cluster and resource pool for the current day
// @Tags Cluster Resources
// @Accept json
// @Produce json
// @Param cluster_name query string true "Cluster name"
// @Param resource_pool query string true "Resource pool name"
// @Success 200 {object} service.ResourcePoolAllocationRateDTO "Successfully retrieved allocation rates"
// @Failure 400 {object} render.ErrorResponse "Missing required parameters"
// @Failure 404 {object} render.ErrorResponse "No data found for the specified cluster and resource pool"
// @Failure 500 {object} render.ErrorResponse "Internal server error"
// @Router /fe-v1/cluster-resources/allocation-rate [get]
type ResourcePoolAllocationQuery struct {
	ClusterName  string `form:"cluster_name" binding:"required"`
	ResourcePool string `form:"resource_pool" binding:"required"`
}

func (h *ClusterResourceHandler) GetResourcePoolAllocationRate(c *gin.Context) {
	var query ResourcePoolAllocationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		render.BadRequest(c, MsgInvalidParams+err.Error())
		return
	}

	// Get allocation rates for today
	allocationRate, err := h.service.GetResourcePoolAllocationRate(query.ClusterName, query.ResourcePool)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, "Failed to get resource pool allocation rate: "+err.Error())
		return
	}

	// Check if data was found
	if allocationRate == nil {
		render.Success(c, nil) // Return empty data for frontend to handle
		return
	}

	render.Success(c, allocationRate)
}
