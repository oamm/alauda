package api

import (
	"context"
	"database/sql"
	"errors"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
)

func (h *incidentHandler) GetIncident(ctx context.Context, req *connect.Request[registryv1.GetIncidentRequest]) (*connect.Response[registryv1.GetIncidentResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	item, err := h.repo.Get(ctx, req.Msg.GetId())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.GetIncidentResponse{Incident: item}), nil
}

func (h *incidentHandler) ListIncidents(ctx context.Context, req *connect.Request[registryv1.ListIncidentsRequest]) (*connect.Response[registryv1.ListIncidentsResponse], error) {
	pageSize := int32(50)
	pageToken := ""
	if req.Msg.GetPagination() != nil {
		if req.Msg.GetPagination().GetPageSize() > 0 {
			pageSize = req.Msg.GetPagination().GetPageSize()
		}
		pageToken = req.Msg.GetPagination().GetPageToken()
	}

	items, nextToken, err := h.repo.List(ctx, req.Msg.GetEnvironmentId(), req.Msg.GetServiceId(), req.Msg.GetDeploymentId(), req.Msg.GetInstanceId(), req.Msg.GetState(), int(pageSize), pageToken)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.ListIncidentsResponse{
		Incidents: items,
		Pagination: &registryv1.PaginationResponse{
			NextPageToken: nextToken,
			TotalSize:     int32(len(items)),
		},
	}), nil
}

func (h *incidentHandler) ResolveIncident(ctx context.Context, req *connect.Request[registryv1.ResolveIncidentRequest]) (*connect.Response[registryv1.ResolveIncidentResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	item, err := h.repo.Resolve(ctx, req.Msg.GetId(), req.Msg.GetReason())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.ResolveIncidentResponse{Incident: item}), nil
}
