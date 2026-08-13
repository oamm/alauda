package api

import (
	"context"
	"database/sql"
	"errors"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
)

func (h *instanceHandler) CreateInstance(ctx context.Context, req *connect.Request[registryv1.CreateInstanceRequest]) (*connect.Response[registryv1.CreateInstanceResponse], error) {
	if req.Msg.GetDeploymentId() == "" || req.Msg.GetName() == "" || req.Msg.GetAddress() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("deployment_id, name, and address are required"))
	}
	item, err := h.repo.Create(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.CreateInstanceResponse{Instance: item}), nil
}

func (h *instanceHandler) GetInstance(ctx context.Context, req *connect.Request[registryv1.GetInstanceRequest]) (*connect.Response[registryv1.GetInstanceResponse], error) {
	item, err := h.repo.Get(ctx, req.Msg.GetId())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.GetInstanceResponse{Instance: item}), nil
}

func (h *instanceHandler) ListInstances(ctx context.Context, req *connect.Request[registryv1.ListInstancesRequest]) (*connect.Response[registryv1.ListInstancesResponse], error) {
	pageSize := int32(50)
	pageToken := ""
	if req.Msg.GetPagination() != nil {
		if req.Msg.GetPagination().GetPageSize() > 0 {
			pageSize = req.Msg.GetPagination().GetPageSize()
		}
		pageToken = req.Msg.GetPagination().GetPageToken()
	}

	items, nextToken, err := h.repo.List(ctx, req.Msg.GetDeploymentId(), int(pageSize), pageToken)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&registryv1.ListInstancesResponse{
		Instances: items,
		Pagination: &registryv1.PaginationResponse{
			NextPageToken: nextToken,
			TotalSize:     int32(len(items)),
		},
	}), nil
}

func (h *instanceHandler) UpdateInstance(ctx context.Context, req *connect.Request[registryv1.UpdateInstanceRequest]) (*connect.Response[registryv1.UpdateInstanceResponse], error) {
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
	return connect.NewResponse(&registryv1.UpdateInstanceResponse{Instance: item}), nil
}

func (h *instanceHandler) DeleteInstance(ctx context.Context, req *connect.Request[registryv1.DeleteInstanceRequest]) (*connect.Response[registryv1.DeleteInstanceResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	if err := h.repo.Delete(ctx, req.Msg.GetId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.DeleteInstanceResponse{}), nil
}
