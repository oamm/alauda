package api

import (
	"context"
	"database/sql"
	"errors"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	healthsvc "github.com/company/service-registry/internal/health"
)

func (h *healthHandler) CreateHealthCheck(ctx context.Context, req *connect.Request[registryv1.CreateHealthCheckRequest]) (*connect.Response[registryv1.CreateHealthCheckResponse], error) {
	if req.Msg.GetInstanceId() == "" || req.Msg.GetName() == "" || req.Msg.GetType() == registryv1.HealthCheckType_HEALTH_CHECK_TYPE_UNSPECIFIED {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("instance_id, name, and type are required"))
	}
	item, err := h.repo.CreateHealthCheck(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.CreateHealthCheckResponse{HealthCheck: item}), nil
}

func (h *healthHandler) GetHealthCheck(ctx context.Context, req *connect.Request[registryv1.GetHealthCheckRequest]) (*connect.Response[registryv1.GetHealthCheckResponse], error) {
	item, err := h.repo.GetHealthCheck(ctx, req.Msg.GetId())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.GetHealthCheckResponse{HealthCheck: item}), nil
}

func (h *healthHandler) ListHealthChecks(ctx context.Context, req *connect.Request[registryv1.ListHealthChecksRequest]) (*connect.Response[registryv1.ListHealthChecksResponse], error) {
	pageSize := int32(50)
	pageToken := ""
	if req.Msg.GetPagination() != nil {
		if req.Msg.GetPagination().GetPageSize() > 0 {
			pageSize = req.Msg.GetPagination().GetPageSize()
		}
		pageToken = req.Msg.GetPagination().GetPageToken()
	}

	items, nextToken, err := h.repo.ListHealthChecks(ctx, req.Msg.GetInstanceId(), req.Msg.GetIncludeDisabled(), int(pageSize), pageToken)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&registryv1.ListHealthChecksResponse{
		HealthChecks: items,
		Pagination: &registryv1.PaginationResponse{
			NextPageToken: nextToken,
			TotalSize:     int32(len(items)),
		},
	}), nil
}

func (h *healthHandler) UpdateHealthCheck(ctx context.Context, req *connect.Request[registryv1.UpdateHealthCheckRequest]) (*connect.Response[registryv1.UpdateHealthCheckResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	item, err := h.repo.UpdateHealthCheck(ctx, req.Msg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.UpdateHealthCheckResponse{HealthCheck: item}), nil
}

func (h *healthHandler) DeleteHealthCheck(ctx context.Context, req *connect.Request[registryv1.DeleteHealthCheckRequest]) (*connect.Response[registryv1.DeleteHealthCheckResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	if err := h.repo.DeleteHealthCheck(ctx, req.Msg.GetId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.DeleteHealthCheckResponse{}), nil
}

func (h *healthHandler) RunHealthCheck(ctx context.Context, req *connect.Request[registryv1.RunHealthCheckRequest]) (*connect.Response[registryv1.RunHealthCheckResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	target, err := h.repo.GetHealthCheckTarget(ctx, req.Msg.GetId())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := h.executor.Execute(ctx, healthsvc.Target{
		Check:   target.Check,
		Address: target.Address,
		Port:    target.Port,
		Path:    target.Path,
	})
	state, err := h.repo.RecordHealthResult(ctx, target.Check, result)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.RunHealthCheckResponse{
		Result: result,
		State:  state,
	}), nil
}

func (h *healthHandler) GetInstanceHealthState(ctx context.Context, req *connect.Request[registryv1.GetInstanceHealthStateRequest]) (*connect.Response[registryv1.GetInstanceHealthStateResponse], error) {
	if req.Msg.GetInstanceId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("instance_id is required"))
	}
	state, err := h.repo.GetInstanceHealthState(ctx, req.Msg.GetInstanceId())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.GetInstanceHealthStateResponse{State: state}), nil
}

func (h *healthHandler) ListHealthResults(ctx context.Context, req *connect.Request[registryv1.ListHealthResultsRequest]) (*connect.Response[registryv1.ListHealthResultsResponse], error) {
	pageSize := int32(50)
	pageToken := ""
	if req.Msg.GetPagination() != nil {
		if req.Msg.GetPagination().GetPageSize() > 0 {
			pageSize = req.Msg.GetPagination().GetPageSize()
		}
		pageToken = req.Msg.GetPagination().GetPageToken()
	}

	items, nextToken, err := h.repo.ListHealthResults(ctx, req.Msg.GetHealthCheckId(), req.Msg.GetInstanceId(), int(pageSize), pageToken)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.ListHealthResultsResponse{
		Results: items,
		Pagination: &registryv1.PaginationResponse{
			NextPageToken: nextToken,
			TotalSize:     int32(len(items)),
		},
	}), nil
}

func (h *healthHandler) GetAvailability(ctx context.Context, req *connect.Request[registryv1.GetAvailabilityRequest]) (*connect.Response[registryv1.GetAvailabilityResponse], error) {
	response, err := h.availabilityRepo.GetAvailability(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(response), nil
}
