package api

import (
	"context"
	"time"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/storage"
)

func (h *eventHandler) ListEvents(ctx context.Context, req *connect.Request[registryv1.ListEventsRequest]) (*connect.Response[registryv1.ListEventsResponse], error) {
	pageSize := int32(50)
	pageToken := ""
	if req.Msg.GetPagination() != nil {
		if req.Msg.GetPagination().GetPageSize() > 0 {
			pageSize = req.Msg.GetPagination().GetPageSize()
		}
		pageToken = req.Msg.GetPagination().GetPageToken()
	}

	items, nextToken, err := h.repo.List(ctx, eventFiltersFromListRequest(req.Msg), int(pageSize), pageToken)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.ListEventsResponse{
		Events: items,
		Pagination: &registryv1.PaginationResponse{
			NextPageToken: nextToken,
			TotalSize:     int32(len(items)),
		},
	}), nil
}

func (h *eventHandler) WatchEvents(ctx context.Context, req *connect.Request[registryv1.WatchEventsRequest], stream *connect.ServerStream[registryv1.Event]) error {
	filters := storage.EventFilters{
		EnvironmentID: req.Msg.GetEnvironmentId(),
		ServiceID:     req.Msg.GetServiceId(),
		DeploymentID:  req.Msg.GetDeploymentId(),
		InstanceID:    req.Msg.GetInstanceId(),
	}
	cursor := time.Now().UTC().Add(-time.Nanosecond)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		events, err := h.repo.ListSince(ctx, filters, cursor, 50)
		if err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
		for _, event := range events {
			if event.GetTimestamp() != nil {
				cursor = event.GetTimestamp().AsTime().UTC()
			}
			if err := stream.Send(event); err != nil {
				return err
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func eventFiltersFromListRequest(req *registryv1.ListEventsRequest) storage.EventFilters {
	return storage.EventFilters{
		EnvironmentID: req.GetEnvironmentId(),
		ServiceID:     req.GetServiceId(),
		DeploymentID:  req.GetDeploymentId(),
		InstanceID:    req.GetInstanceId(),
		Type:          req.GetType(),
	}
}
