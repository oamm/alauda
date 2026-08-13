package api

import (
	"context"
	"database/sql"
	"errors"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/alerts"
)

func (h *alertHandler) CreateNotificationChannel(ctx context.Context, req *connect.Request[registryv1.CreateNotificationChannelRequest]) (*connect.Response[registryv1.CreateNotificationChannelResponse], error) {
	if req.Msg.GetType() == "" || req.Msg.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("type and name are required"))
	}
	item, err := h.repo.CreateNotificationChannel(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.CreateNotificationChannelResponse{Channel: item}), nil
}

func (h *alertHandler) GetNotificationChannel(ctx context.Context, req *connect.Request[registryv1.GetNotificationChannelRequest]) (*connect.Response[registryv1.GetNotificationChannelResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	item, err := h.repo.GetNotificationChannel(ctx, req.Msg.GetId())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.GetNotificationChannelResponse{Channel: item}), nil
}

func (h *alertHandler) ListNotificationChannels(ctx context.Context, req *connect.Request[registryv1.ListNotificationChannelsRequest]) (*connect.Response[registryv1.ListNotificationChannelsResponse], error) {
	pageSize, pageToken := pagination(req.Msg.GetPagination())
	items, nextToken, err := h.repo.ListNotificationChannels(ctx, int(pageSize), pageToken)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.ListNotificationChannelsResponse{
		Channels: items,
		Pagination: &registryv1.PaginationResponse{
			NextPageToken: nextToken,
			TotalSize:     int32(len(items)),
		},
	}), nil
}

func (h *alertHandler) UpdateNotificationChannel(ctx context.Context, req *connect.Request[registryv1.UpdateNotificationChannelRequest]) (*connect.Response[registryv1.UpdateNotificationChannelResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	item, err := h.repo.UpdateNotificationChannel(ctx, req.Msg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.UpdateNotificationChannelResponse{Channel: item}), nil
}

func (h *alertHandler) DeleteNotificationChannel(ctx context.Context, req *connect.Request[registryv1.DeleteNotificationChannelRequest]) (*connect.Response[registryv1.DeleteNotificationChannelResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	if err := h.repo.DeleteNotificationChannel(ctx, req.Msg.GetId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.DeleteNotificationChannelResponse{}), nil
}

func (h *alertHandler) TestNotificationChannel(ctx context.Context, req *connect.Request[registryv1.TestNotificationChannelRequest]) (*connect.Response[registryv1.TestNotificationChannelResponse], error) {
	if req.Msg.GetChannelId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("channel_id is required"))
	}
	channel, err := h.repo.GetNotificationChannel(ctx, req.Msg.GetChannelId())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	engine := h.engine
	if engine == nil {
		engine = alerts.NewEngine(h.repo, nil)
	}
	if err := engine.TestChannel(ctx, channel); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(&registryv1.TestNotificationChannelResponse{
		Success: true,
		Message: "sent",
	}), nil
}

func (h *alertHandler) CreateAlertPolicy(ctx context.Context, req *connect.Request[registryv1.CreateAlertPolicyRequest]) (*connect.Response[registryv1.CreateAlertPolicyResponse], error) {
	if req.Msg.GetDeploymentId() == "" && req.Msg.GetEnvironmentId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("deployment_id or environment_id is required"))
	}
	item, err := h.repo.CreateAlertPolicy(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.CreateAlertPolicyResponse{Policy: item}), nil
}

func (h *alertHandler) GetAlertPolicy(ctx context.Context, req *connect.Request[registryv1.GetAlertPolicyRequest]) (*connect.Response[registryv1.GetAlertPolicyResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	item, err := h.repo.GetAlertPolicy(ctx, req.Msg.GetId())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.GetAlertPolicyResponse{Policy: item}), nil
}

func (h *alertHandler) ListAlertPolicies(ctx context.Context, req *connect.Request[registryv1.ListAlertPoliciesRequest]) (*connect.Response[registryv1.ListAlertPoliciesResponse], error) {
	pageSize, pageToken := pagination(req.Msg.GetPagination())
	items, nextToken, err := h.repo.ListAlertPolicies(ctx, req.Msg.GetDeploymentId(), req.Msg.GetEnvironmentId(), int(pageSize), pageToken)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.ListAlertPoliciesResponse{
		Policies: items,
		Pagination: &registryv1.PaginationResponse{
			NextPageToken: nextToken,
			TotalSize:     int32(len(items)),
		},
	}), nil
}

func (h *alertHandler) UpdateAlertPolicy(ctx context.Context, req *connect.Request[registryv1.UpdateAlertPolicyRequest]) (*connect.Response[registryv1.UpdateAlertPolicyResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	item, err := h.repo.UpdateAlertPolicy(ctx, req.Msg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.UpdateAlertPolicyResponse{Policy: item}), nil
}

func (h *alertHandler) DeleteAlertPolicy(ctx context.Context, req *connect.Request[registryv1.DeleteAlertPolicyRequest]) (*connect.Response[registryv1.DeleteAlertPolicyResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	if err := h.repo.DeleteAlertPolicy(ctx, req.Msg.GetId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.DeleteAlertPolicyResponse{}), nil
}

func pagination(req *registryv1.PaginationRequest) (int32, string) {
	pageSize := int32(50)
	pageToken := ""
	if req != nil {
		if req.GetPageSize() > 0 {
			pageSize = req.GetPageSize()
		}
		pageToken = req.GetPageToken()
	}
	return pageSize, pageToken
}
