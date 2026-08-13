package api

import (
	"context"
	"database/sql"
	"errors"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
)

func (h *deploymentHandler) CreateDeployment(ctx context.Context, req *connect.Request[registryv1.CreateDeploymentRequest]) (*connect.Response[registryv1.CreateDeploymentResponse], error) {
	if req.Msg.GetServiceId() == "" || req.Msg.GetEnvironmentId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("service_id and environment_id are required"))
	}
	item, err := h.repo.Create(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.CreateDeploymentResponse{Deployment: item}), nil
}

func (h *deploymentHandler) GetDeployment(ctx context.Context, req *connect.Request[registryv1.GetDeploymentRequest]) (*connect.Response[registryv1.GetDeploymentResponse], error) {
	item, err := h.repo.Get(ctx, req.Msg.GetId())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.GetDeploymentResponse{Deployment: item}), nil
}

func (h *deploymentHandler) ListDeployments(ctx context.Context, req *connect.Request[registryv1.ListDeploymentsRequest]) (*connect.Response[registryv1.ListDeploymentsResponse], error) {
	pageSize := int32(50)
	pageToken := ""
	if req.Msg.GetPagination() != nil {
		if req.Msg.GetPagination().GetPageSize() > 0 {
			pageSize = req.Msg.GetPagination().GetPageSize()
		}
		pageToken = req.Msg.GetPagination().GetPageToken()
	}

	items, nextToken, err := h.repo.List(ctx, req.Msg.GetServiceId(), req.Msg.GetEnvironmentId(), int(pageSize), pageToken)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&registryv1.ListDeploymentsResponse{
		Deployments: items,
		Pagination: &registryv1.PaginationResponse{
			NextPageToken: nextToken,
			TotalSize:     int32(len(items)),
		},
	}), nil
}

func (h *deploymentHandler) UpdateDeployment(ctx context.Context, req *connect.Request[registryv1.UpdateDeploymentRequest]) (*connect.Response[registryv1.UpdateDeploymentResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	item, err := h.repo.Update(ctx, req.Msg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.UpdateDeploymentResponse{Deployment: item}), nil
}

func (h *deploymentHandler) DeleteDeployment(ctx context.Context, req *connect.Request[registryv1.DeleteDeploymentRequest]) (*connect.Response[registryv1.DeleteDeploymentResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	if err := h.repo.Delete(ctx, req.Msg.GetId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.DeleteDeploymentResponse{}), nil
}
