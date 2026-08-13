package api

import (
	"context"
	"database/sql"
	"errors"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
)

func (h *endpointHandler) CreateEndpoint(ctx context.Context, req *connect.Request[registryv1.CreateEndpointRequest]) (*connect.Response[registryv1.CreateEndpointResponse], error) {
	if req.Msg.GetInstanceId() == "" || req.Msg.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("instance_id and name are required"))
	}
	item, err := h.repo.Create(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.CreateEndpointResponse{Endpoint: item}), nil
}

func (h *endpointHandler) GetEndpoint(ctx context.Context, req *connect.Request[registryv1.GetEndpointRequest]) (*connect.Response[registryv1.GetEndpointResponse], error) {
	item, err := h.repo.Get(ctx, req.Msg.GetId())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.GetEndpointResponse{Endpoint: item}), nil
}

func (h *endpointHandler) ListEndpoints(ctx context.Context, req *connect.Request[registryv1.ListEndpointsRequest]) (*connect.Response[registryv1.ListEndpointsResponse], error) {
	pageSize := int32(50)
	pageToken := ""
	if req.Msg.GetPagination() != nil {
		if req.Msg.GetPagination().GetPageSize() > 0 {
			pageSize = req.Msg.GetPagination().GetPageSize()
		}
		pageToken = req.Msg.GetPagination().GetPageToken()
	}

	items, nextToken, err := h.repo.List(ctx, req.Msg.GetInstanceId(), int(pageSize), pageToken)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&registryv1.ListEndpointsResponse{
		Endpoints: items,
		Pagination: &registryv1.PaginationResponse{
			NextPageToken: nextToken,
			TotalSize:     int32(len(items)),
		},
	}), nil
}

func (h *endpointHandler) UpdateEndpoint(ctx context.Context, req *connect.Request[registryv1.UpdateEndpointRequest]) (*connect.Response[registryv1.UpdateEndpointResponse], error) {
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
	return connect.NewResponse(&registryv1.UpdateEndpointResponse{Endpoint: item}), nil
}

func (h *endpointHandler) DeleteEndpoint(ctx context.Context, req *connect.Request[registryv1.DeleteEndpointRequest]) (*connect.Response[registryv1.DeleteEndpointResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	if err := h.repo.Delete(ctx, req.Msg.GetId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.DeleteEndpointResponse{}), nil
}
