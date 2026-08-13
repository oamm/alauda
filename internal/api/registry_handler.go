package api

import (
	"context"
	"database/sql"
	"errors"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
)

func (h *registryHandler) ResolveService(ctx context.Context, req *connect.Request[registryv1.ResolveServiceRequest]) (*connect.Response[registryv1.ResolveServiceResponse], error) {
	if req.Msg.GetServiceName() == "" || req.Msg.GetEnvironmentKey() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("service_name and environment_key are required"))
	}
	serviceID, deploymentID, endpoints, err := h.repo.ResolveService(ctx, req.Msg.GetServiceName(), req.Msg.GetEnvironmentKey(), req.Msg.GetHealthyOnly())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.ResolveServiceResponse{
		ServiceId:    serviceID,
		DeploymentId: deploymentID,
		Endpoints:    endpoints,
	}), nil
}

func (h *registryHandler) ResolveEndpoint(ctx context.Context, req *connect.Request[registryv1.ResolveEndpointRequest]) (*connect.Response[registryv1.ResolveEndpointResponse], error) {
	if req.Msg.GetServiceName() == "" || req.Msg.GetEndpointName() == "" || req.Msg.GetEnvironmentKey() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("service_name, endpoint_name and environment_key are required"))
	}
	serviceID, deploymentID, endpoints, err := h.repo.ResolveEndpoint(ctx, req.Msg.GetServiceName(), req.Msg.GetEndpointName(), req.Msg.GetEnvironmentKey(), req.Msg.GetHealthyOnly())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.ResolveEndpointResponse{
		ServiceId:    serviceID,
		DeploymentId: deploymentID,
		Endpoints:    endpoints,
	}), nil
}
