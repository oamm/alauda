package api

import (
	"context"
	"database/sql"
	"errors"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
)

func (h *catalogHandler) CreateService(ctx context.Context, req *connect.Request[registryv1.CreateServiceRequest]) (*connect.Response[registryv1.CreateServiceResponse], error) {
	if req.Msg.GetName() == "" || req.Msg.GetDisplayName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name and display_name are required"))
	}
	svc, err := h.repo.Create(ctx, req.Msg)
	if err != nil {
		return nil, safeConnectError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.CreateServiceResponse{Service: svc}), nil
}

func (h *catalogHandler) GetService(ctx context.Context, req *connect.Request[registryv1.GetServiceRequest]) (*connect.Response[registryv1.GetServiceResponse], error) {
	svc, err := h.repo.Get(ctx, req.Msg.GetId())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, safeConnectError(connect.CodeNotFound, err)
		}
		return nil, safeConnectError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.GetServiceResponse{Service: svc}), nil
}

func (h *catalogHandler) ListServices(ctx context.Context, req *connect.Request[registryv1.ListServicesRequest]) (*connect.Response[registryv1.ListServicesResponse], error) {
	pageSize := int32(50)
	pageToken := ""
	if req.Msg.GetPagination() != nil {
		if req.Msg.GetPagination().GetPageSize() > 0 {
			pageSize = req.Msg.GetPagination().GetPageSize()
		}
		pageToken = req.Msg.GetPagination().GetPageToken()
	}

	items, nextToken, err := h.repo.List(ctx, req.Msg.GetEnvironmentId(), int(pageSize), pageToken)
	if err != nil {
		return nil, safeConnectError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&registryv1.ListServicesResponse{
		Services: items,
		Pagination: &registryv1.PaginationResponse{
			NextPageToken: nextToken,
			TotalSize:     int32(len(items)),
		},
	}), nil
}

func (h *catalogHandler) UpdateService(ctx context.Context, req *connect.Request[registryv1.UpdateServiceRequest]) (*connect.Response[registryv1.UpdateServiceResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	svc, err := h.repo.Update(ctx, req.Msg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, safeConnectError(connect.CodeNotFound, err)
		}
		return nil, safeConnectError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.UpdateServiceResponse{Service: svc}), nil
}

func (h *catalogHandler) DeleteService(ctx context.Context, req *connect.Request[registryv1.DeleteServiceRequest]) (*connect.Response[registryv1.DeleteServiceResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	if err := h.repo.Delete(ctx, req.Msg.GetId()); err != nil {
		return nil, safeConnectError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.DeleteServiceResponse{}), nil
}
