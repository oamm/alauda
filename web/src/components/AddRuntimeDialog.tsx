import { FormEvent, useEffect, useRef, useState } from "react";

import { Environment, Service } from "../api";
import { Button, IconButton } from "./OperationsUI";

export type RegistrationEndpointDraft = {
  name: string;
  protocol: string;
  port: number;
  path: string;
  primary: boolean;
};

export type RegistrationFormState = {
  environmentId: string;
  instanceName: string;
  address: string;
  description: string;
  endpoints: RegistrationEndpointDraft[];
  configureHealth: boolean;
  healthName: string;
  healthType: string;
  healthPath: string;
  healthIntervalSeconds: number;
  healthTimeoutSeconds: number;
  healthFailuresBeforeUnhealthy: number;
  healthSuccessesBeforeHealthy: number;
};

type Props = {
  environments: Environment[];
  form: RegistrationFormState;
  registrationSuccess: string;
  saving: boolean;
  selectedService: Service;
  onAddEndpoint: () => void;
  onRemoveEndpoint: (index: number) => void;
  onSetPrimaryEndpoint: (index: number) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onUpdateEndpoint: (index: number, updates: Partial<RegistrationEndpointDraft>) => void;
  setForm: React.Dispatch<React.SetStateAction<RegistrationFormState>>;
  onClose: () => void;
  registeredRuntime?: {
    name: string;
    address: string;
    endpoints: Array<Pick<RegistrationEndpointDraft, "protocol" | "port" | "path" | "primary">>;
  };
  healthError: string;
  healthConfigured: boolean;
  healthSaving: boolean;
  onConfigureHealth: () => void;
  onCreateHealth: () => void;
};

export function AddRuntimeDialog({
  environments, form, registrationSuccess, saving, selectedService,
  onAddEndpoint, onRemoveEndpoint, onSetPrimaryEndpoint, onSubmit,
  onUpdateEndpoint, setForm, onClose, registeredRuntime, healthError,
  healthConfigured, healthSaving, onConfigureHealth, onCreateHealth,
}: Props) {
  const dialogRef = useRef<HTMLDivElement>(null);
  const [healthOpen, setHealthOpen] = useState(false);
  useEffect(() => {
    dialogRef.current?.querySelector<HTMLInputElement>("[data-registration-instance-name]")?.focus();
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onClose();
      }
    };
    window.addEventListener("keydown", handleEscape);
    return () => window.removeEventListener("keydown", handleEscape);
  }, [onClose]);

  return <div className="modal-backdrop" role="presentation">
    <div aria-labelledby="add-instance-title" aria-modal="true" className="modal form-panel drawer-form runtime-registration-dialog" ref={dialogRef} role="dialog">
      <form noValidate onSubmit={onSubmit}>
        <div className="modal-header">
          <div><h2 id="add-instance-title">Add instance</h2><span>{selectedService.displayName || selectedService.name}</span></div>
          <IconButton label="Close dialog" type="button" onClick={onClose}>x</IconButton>
        </div>
        <div className="modal-body">
        <div className="context-strip"><span>Service</span><strong>{selectedService.displayName || selectedService.name}</strong></div>
        {registeredRuntime ? <section className="form-section" aria-live="polite">
          <p className="success" role="status">Instance added successfully</p>
          <h3>{registeredRuntime.name}</h3>
          <p>{registeredRuntime.address}</p>
          <h4>Endpoints</h4>
          <ul>
            {registeredRuntime.endpoints.map((endpoint, index) => <li key={`${endpoint.protocol}-${endpoint.port}-${index}`}>
              {endpoint.protocol.replace("PROTOCOL_", "").toUpperCase()} :{endpoint.port}{endpoint.path || ""}{endpoint.primary ? " (Primary)" : ""}
            </li>)}
          </ul>
          <h4>Health monitoring</h4>
          {healthError ? <p className="error" role="alert">Health monitoring could not be configured. The instance and endpoints were created successfully.</p> : <p>{healthConfigured ? "Enabled" : "Not configured"}</p>}
          {!healthOpen ? <div className="drawer-actions">
            <button type="button" onClick={() => { setHealthOpen(true); onConfigureHealth(); }}>{healthError ? "Retry health configuration" : "Configure health monitoring"}</button>
            <button type="button" onClick={onClose}>Done</button>
          </div> : <div className="form-fields-grid">
            <label>Check name<input value={form.healthName} onChange={(event) => setForm((current) => ({ ...current, healthName: event.target.value }))} /></label>
            <label>Check type<select value={form.healthType} onChange={(event) => setForm((current) => ({ ...current, healthType: event.target.value }))}><option value="HEALTH_CHECK_TYPE_HTTP">HTTP</option><option value="HEALTH_CHECK_TYPE_HTTPS">HTTPS</option><option value="HEALTH_CHECK_TYPE_GRPC">gRPC</option><option value="HEALTH_CHECK_TYPE_TCP">TCP</option><option value="HEALTH_CHECK_TYPE_UDP">UDP</option></select></label>
            <label>Interval (seconds)<input min="1" type="number" value={form.healthIntervalSeconds} onChange={(event) => setForm((current) => ({ ...current, healthIntervalSeconds: Number(event.target.value) }))} /></label>
            <label>Timeout (seconds)<input min="1" type="number" value={form.healthTimeoutSeconds} onChange={(event) => setForm((current) => ({ ...current, healthTimeoutSeconds: Number(event.target.value) }))} /></label>
            <div className="drawer-actions"><button disabled={healthSaving} type="button" onClick={onCreateHealth}>{healthSaving ? "Saving" : healthError ? "Retry health configuration" : "Save health monitoring"}</button><button type="button" onClick={onClose}>Done</button></div>
          </div>}
        </section> : <>
        <section className="form-section">
          <div className="form-section-heading"><span>Required</span></div>
          <label>Environment<select aria-label="Environment" required value={form.environmentId} onChange={(event) => setForm((current) => ({ ...current, environmentId: event.target.value }))}><option value="">Select environment</option>{environments.map((environment) => <option key={environment.id} value={environment.id}>{environment.name}</option>)}</select></label>
          <div className="form-fields-grid">
            <label>Instance name<input aria-label="Instance name" data-registration-instance-name required value={form.instanceName} onChange={(event) => setForm((current) => ({ ...current, instanceName: event.target.value }))} /></label>
            <label>Address / hostname<input aria-label="Address / hostname" required value={form.address} onChange={(event) => setForm((current) => ({ ...current, address: event.target.value }))} /></label>
          </div>
          <label>Description <span className="field-hint">Optional</span><input value={form.description} onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))} /></label>
        </section>
        <section className="form-section">
          <div className="form-section-heading"><span>Endpoints</span><strong>{form.endpoints.length} configured</strong></div>
          <p>Expose one or more endpoints for this service address. Choose one primary endpoint.</p>
          <div className="endpoint-editor">
            {form.endpoints.map((endpoint, index) => <div className="endpoint-editor-row" key={index}>
              <label>Protocol<select aria-label={`Endpoint ${index + 1} protocol`} value={endpoint.protocol} onChange={(event) => onUpdateEndpoint(index, { protocol: event.target.value })}><option value="PROTOCOL_HTTP">HTTP</option><option value="PROTOCOL_HTTPS">HTTPS</option><option value="PROTOCOL_GRPC">gRPC</option><option value="PROTOCOL_TCP">TCP</option><option value="PROTOCOL_UDP">UDP</option></select></label>
              <label>Port<input aria-label={`Endpoint ${index + 1} port`} max="65535" min="1" required type="number" value={endpoint.port} onChange={(event) => onUpdateEndpoint(index, { port: Number(event.target.value) })} /></label>
              <label>Path<input aria-label={`Endpoint ${index + 1} path`} value={endpoint.path} onChange={(event) => onUpdateEndpoint(index, { path: event.target.value })} /></label>
              <label className="checkbox-label"><input aria-label={`Endpoint ${index + 1} primary`} checked={endpoint.primary} name="registration-primary-endpoint" type="radio" onChange={() => onSetPrimaryEndpoint(index)} /> Primary</label>
              <IconButton disabled={form.endpoints.length === 1} label="Remove endpoint" size="sm" type="button" onClick={() => onRemoveEndpoint(index)}>x</IconButton>
            </div>)}
          </div>
          <Button size="sm" type="button" onClick={onAddEndpoint}>+ Add endpoint</Button>
        </section>
        {registrationSuccess ? <p className="success" role="status">{registrationSuccess}</p> : null}
        <div className="drawer-actions"><button type="button" onClick={onClose}>Cancel</button><button disabled={saving || environments.length === 0} type="submit">{saving ? "Adding" : "Add instance"}</button></div>
        </>}
        </div>
      </form>
    </div>
  </div>;
}
