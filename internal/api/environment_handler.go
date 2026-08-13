package api

import (
	"context"
	"database/sql"
	"errors"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
)

func (h *environmentHandler) CreateEnvironment(ctx context.Context, req *connect.Request[registryv1.CreateEnvironmentRequest]) (*connect.Response[registryv1.CreateEnvironmentResponse], error) {
	if req.Msg.GetKey() == "" || req.Msg.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("key and name are required"))
	}
	env, err := h.repo.Create(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.CreateEnvironmentResponse{Environment: env}), nil
}

func (h *environmentHandler) GetEnvironment(ctx context.Context, req *connect.Request[registryv1.GetEnvironmentRequest]) (*connect.Response[registryv1.GetEnvironmentResponse], error) {
	env, err := h.repo.Get(ctx, req.Msg.GetId())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.GetEnvironmentResponse{Environment: env}), nil
}

func (h *environmentHandler) ListEnvironments(ctx context.Context, req *connect.Request[registryv1.ListEnvironmentsRequest]) (*connect.Response[registryv1.ListEnvironmentsResponse], error) {
	pageSize := int32(50)
	pageToken := ""
	if req.Msg.GetPagination() != nil {
		if req.Msg.GetPagination().GetPageSize() > 0 {
			pageSize = req.Msg.GetPagination().GetPageSize()
		}
		pageToken = req.Msg.GetPagination().GetPageToken()
	}

	items, nextToken, err := h.repo.List(ctx, req.Msg.GetIncludeDisabled(), int(pageSize), pageToken)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&registryv1.ListEnvironmentsResponse{
		Environments: items,
		Pagination: &registryv1.PaginationResponse{
			NextPageToken: nextToken,
			TotalSize:     int32(len(items)),
		},
	}), nil
}

func (h *environmentHandler) UpdateEnvironment(ctx context.Context, req *connect.Request[registryv1.UpdateEnvironmentRequest]) (*connect.Response[registryv1.UpdateEnvironmentResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	env, err := h.repo.Update(ctx, req.Msg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.UpdateEnvironmentResponse{Environment: env}), nil
}

func (h *environmentHandler) DeleteEnvironment(ctx context.Context, req *connect.Request[registryv1.DeleteEnvironmentRequest]) (*connect.Response[registryv1.DeleteEnvironmentResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	if err := h.repo.Delete(ctx, req.Msg.GetId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.DeleteEnvironmentResponse{}), nil
}
