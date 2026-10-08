package api

import (
	"context"
	"database/sql"
	"errors"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	healthsvc "github.com/company/service-registry/internal/health"
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

func (h *incidentHandler) VerifyIncidentRecovery(ctx context.Context, req *connect.Request[registryv1.VerifyIncidentRecoveryRequest]) (*connect.Response[registryv1.VerifyIncidentRecoveryResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	incident, err := h.repo.Get(ctx, req.Msg.GetId())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if incident.GetState() != registryv1.IncidentState_INCIDENT_STATE_OPEN {
		return connect.NewResponse(&registryv1.VerifyIncidentRecoveryResponse{
			Incident: incident, Recovered: true, CheckedAt: incident.GetResolvedAt(), Reason: "incident already resolved",
		}), nil
	}
	checkID := incident.GetMetadata()["health_check_id"]
	if checkID == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("incident recovery cannot be verified without an originating health check"))
	}
	target, err := h.healthRepo.GetHealthCheckTarget(ctx, checkID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("originating health check is unavailable for verification"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	result := h.executor.Execute(ctx, healthsvc.Target{Check: target.Check, Address: target.Address, Port: target.Port, Path: target.Path})
	if _, err := h.healthRepo.RecordHealthResultForIncident(ctx, target.Check, result, incident.GetId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	updated, err := h.repo.Get(ctx, incident.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	reason := result.GetErrorMessage()
	if reason == "" {
		reason = result.GetErrorType()
	}
	return connect.NewResponse(&registryv1.VerifyIncidentRecoveryResponse{
		Incident: updated, Recovered: result.GetSuccess() && updated.GetState() == registryv1.IncidentState_INCIDENT_STATE_RESOLVED,
		CheckedAt: result.GetTimestamp(), Result: result, Reason: reason,
	}), nil
}

func (h *incidentHandler) ResolveIncidentManually(ctx context.Context, req *connect.Request[registryv1.ResolveIncidentManuallyRequest]) (*connect.Response[registryv1.ResolveIncidentManuallyResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	if req.Msg.GetNote() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("note is required"))
	}
	item, err := h.repo.ResolveManually(ctx, req.Msg.GetId(), req.Msg.GetNote())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.ResolveIncidentManuallyResponse{Incident: item}), nil
}
