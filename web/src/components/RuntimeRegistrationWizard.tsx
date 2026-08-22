import { Dispatch, FormEvent, SetStateAction, useState } from "react";

import { Environment, Service } from "../api";

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

type RuntimeRegistrationWizardProps = {
  environments: Environment[];
  form: RegistrationFormState;
  registrationSuccess: string;
  saving: boolean;
  selectedEnvironment?: Environment;
  selectedService?: Service;
  selectedServiceId: string;
  services: Service[];
  step: number;
  onAddEndpoint: () => void;
  onBack: () => void;
  onRemoveEndpoint: (index: number) => void;
  onSetPrimaryEndpoint: (index: number) => void;
  onServiceChange: (serviceId: string) => void;
  onStepChange: (step: number) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onUpdateEndpoint: (
    index: number,
    updates: Partial<RegistrationEndpointDraft>,
  ) => void;
  setForm: Dispatch<SetStateAction<RegistrationFormState>>;
};

const steps = [
  { id: 0, label: "Target" },
  { id: 1, label: "Instance" },
  { id: 2, label: "Endpoint" },
  { id: 3, label: "Health" },
];

export function RuntimeRegistrationWizard({
  environments,
  form,
  registrationSuccess,
  saving,
  selectedEnvironment,
  selectedService,
  selectedServiceId,
  services,
  step,
  onAddEndpoint,
  onBack,
  onRemoveEndpoint,
  onSetPrimaryEndpoint,
  onServiceChange,
  onStepChange,
  onSubmit,
  onUpdateEndpoint,
  setForm,
}: RuntimeRegistrationWizardProps) {
  const [validationMessage, setValidationMessage] = useState("");
  const readiness = getRegistrationReadiness(form, Boolean(selectedService));
  const canContinue = readiness[step];
  const firstInvalidStep = readiness.findIndex((ready) => !ready);
  const summaryEndpoint = form.endpoints.find((endpoint) => endpoint.primary);
  const currentStep = steps[step];
  const showValidation = Boolean(validationMessage);

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    if (step < steps.length - 1) {
      event.preventDefault();
      if (canContinue) {
        setValidationMessage("");
        onStepChange(Math.min(step + 1, steps.length - 1));
      } else {
        setValidationMessage(stepHelp[step]);
      }
      return;
    }

    if (firstInvalidStep !== -1) {
      event.preventDefault();
      setValidationMessage(stepHelp[firstInvalidStep]);
      onStepChange(firstInvalidStep);
      return;
    }

    setValidationMessage("");
    onSubmit(event);
  }

  function handleStepClick(targetStep: number) {
    const blockingStep = readiness
      .slice(0, targetStep)
      .findIndex((ready) => !ready);
    if (blockingStep === -1) {
      setValidationMessage("");
      onStepChange(targetStep);
      return;
    }
    setValidationMessage(stepHelp[blockingStep]);
    onStepChange(blockingStep);
  }

  return (
    <form
      className="panel form-panel service-wide-panel wizard-panel"
      noValidate
      onSubmit={handleSubmit}
    >
      <div className="wizard-shell">
        <div className="wizard-header">
          <div>
            <h2>Register Runtime</h2>
            <span>
              {selectedService ? selectedService.name : "Select service"} /{" "}
              {selectedEnvironment?.name ?? "Select environment"}
            </span>
          </div>
          <strong>{currentStep?.label}</strong>
        </div>

        <WizardStepper
          currentStep={step}
          readiness={readiness}
          onStepClick={handleStepClick}
        />

        <div className="wizard-stage">
          {step === 0 ? (
            <div className="wizard-step-content">
              <div className="wizard-copy">
                <span>Step 1 of 4</span>
                <h3>Choose where this runtime belongs</h3>
                <p>Select the service and environment for this runtime.</p>
              </div>
              <div className="form-fields-grid">
                <label
                  className={
                    showValidation && !selectedService
                      ? "field-needs-value"
                      : ""
                  }
                >
                  Service
                  <select
                    aria-label="Service"
                    value={selectedServiceId}
                    onChange={(event) => {
                      setValidationMessage("");
                      onServiceChange(event.target.value);
                    }}
                  >
                    <option value="">Select service</option>
                    {services.map((service) => (
                      <option key={service.id} value={service.id}>
                        {service.displayName || service.name}
                      </option>
                    ))}
                  </select>
                  {!selectedService ? (
                    <small className="field-hint">
                      Choose the service this runtime belongs to.
                    </small>
                  ) : (
                    <small className="field-hint success-hint">
                      Preselected from current service context.
                    </small>
                  )}
                </label>
                <label
                  className={
                    showValidation && !form.environmentId
                      ? "field-needs-value"
                      : ""
                  }
                >
                  Environment
                  <select
                    aria-label="Environment"
                    value={form.environmentId}
                    onChange={(event) => {
                      setValidationMessage("");
                      setForm((current) => ({
                        ...current,
                        environmentId: event.target.value,
                      }));
                    }}
                  >
                    <option value="">Select environment</option>
                    {environments.map((environment) => (
                      <option key={environment.id} value={environment.id}>
                        {environment.name}
                      </option>
                    ))}
                  </select>
                  {!form.environmentId ? (
                    <small className="field-hint">
                      Pick the environment where it is running.
                    </small>
                  ) : (
                    <small className="field-hint success-hint">
                      Uses the active environment scope when selected.
                    </small>
                  )}
                </label>
              </div>
            </div>
          ) : null}

          {step === 1 ? (
            <div className="wizard-step-content">
              <div className="wizard-copy">
                <span>Step 2 of 4</span>
                <h3>Name the instance and address</h3>
                <p>Enter the concrete runtime target that Alauda can reach.</p>
              </div>
              <div className="form-fields-grid">
                <label
                  className={
                    showValidation && !form.instanceName.trim()
                      ? "field-needs-value"
                      : ""
                  }
                >
                  Instance name
                  <input
                    aria-label="Instance name"
                    data-registration-instance-name
                    placeholder="payment-api-01"
                    value={form.instanceName}
                    onChange={(event) => {
                      setValidationMessage("");
                      setForm((current) => ({
                        ...current,
                        instanceName: event.target.value,
                      }));
                    }}
                  />
                  {!form.instanceName.trim() ? (
                    <small className="field-hint">
                      Use a stable name operators will recognize.
                    </small>
                  ) : null}
                </label>
                <label
                  className={
                    showValidation && !form.address.trim()
                      ? "field-needs-value"
                      : ""
                  }
                >
                  Address
                  <input
                    aria-label="Address"
                    placeholder="10.0.0.10 or checkout.internal"
                    value={form.address}
                    onChange={(event) => {
                      setValidationMessage("");
                      setForm((current) => ({
                        ...current,
                        address: event.target.value,
                      }));
                    }}
                  />
                  {!form.address.trim() ? (
                    <small className="field-hint">
                      Enter a reachable host, DNS name, or IP.
                    </small>
                  ) : null}
                </label>
                <label className="wide-field">
                  Description
                  <input
                    value={form.description}
                    onChange={(event) =>
                      setForm((current) => ({
                        ...current,
                        description: event.target.value,
                      }))
                    }
                  />
                </label>
              </div>
            </div>
          ) : null}

          {step === 2 ? (
            <div className="wizard-step-content">
              <div className="section-heading">
                <div className="wizard-copy">
                  <span>Step 3 of 4</span>
                  <h3>Define the reachable endpoint</h3>
                  <p>Describe the protocol, port, and path clients should use.</p>
                </div>
                <button type="button" onClick={onAddEndpoint}>
                  Add endpoint
                </button>
              </div>
              <div className="endpoint-editor">
                {form.endpoints.map((endpoint, index) => (
                  <div className="endpoint-editor-row" key={index}>
                    <label
                      className={
                        showValidation && !endpoint.name.trim()
                          ? "field-needs-value"
                          : ""
                      }
                    >
                      Name
                      <input
                        aria-label={`Endpoint ${index + 1} name`}
                        value={endpoint.name}
                        onChange={(event) => {
                          setValidationMessage("");
                          onUpdateEndpoint(index, {
                            name: event.target.value,
                          });
                        }}
                      />
                    </label>
                    <label>
                      Protocol
                      <select
                        aria-label={`Endpoint ${index + 1} protocol`}
                        value={endpoint.protocol}
                        onChange={(event) => {
                          setValidationMessage("");
                          onUpdateEndpoint(index, {
                            protocol: event.target.value,
                          });
                        }}
                      >
                        <option value="PROTOCOL_HTTP">HTTP</option>
                        <option value="PROTOCOL_HTTPS">HTTPS</option>
                        <option value="PROTOCOL_GRPC">gRPC</option>
                        <option value="PROTOCOL_TCP">TCP</option>
                        <option value="PROTOCOL_UDP">UDP</option>
                      </select>
                    </label>
                    <label>
                      Port
                      <input
                        aria-label={`Endpoint ${index + 1} port`}
                        max="65535"
                        min="1"
                        type="number"
                        value={endpoint.port}
                        onChange={(event) => {
                          setValidationMessage("");
                          onUpdateEndpoint(index, {
                            port: Number(event.target.value),
                          });
                        }}
                      />
                    </label>
                    <label>
                      Path
                      <input
                        aria-label={`Endpoint ${index + 1} path`}
                        value={endpoint.path}
                        onChange={(event) => {
                          setValidationMessage("");
                          onUpdateEndpoint(index, {
                            path: event.target.value,
                          });
                        }}
                      />
                    </label>
                    <label className="checkbox-label">
                      <input
                        checked={endpoint.primary}
                        name="registration-primary-endpoint"
                        type="radio"
                        onChange={() => {
                          setValidationMessage("");
                          onSetPrimaryEndpoint(index);
                        }}
                      />
                      Primary
                    </label>
                    <button
                      disabled={form.endpoints.length === 1}
                      type="button"
                      onClick={() => onRemoveEndpoint(index)}
                    >
                      Remove
                    </button>
                  </div>
                ))}
                {!readiness[2] ? (
                  <p className="step-note">
                    Each endpoint needs a name and valid port. One endpoint
                    must be primary.
                  </p>
                ) : null}
              </div>
            </div>
          ) : null}

          {step === 3 ? (
            <div className="wizard-step-content">
              <div className="wizard-copy">
                <span>Step 4 of 4</span>
                <h3>Health check and review</h3>
                <p>Optionally attach monitoring, then review the registration.</p>
              </div>
              <div className="health-config">
                <label className="checkbox-label">
                  <input
                    checked={form.configureHealth}
                    type="checkbox"
                    onChange={(event) => {
                      setValidationMessage("");
                      setForm((current) => ({
                        ...current,
                        configureHealth: event.target.checked,
                      }));
                    }}
                  />
                  Add health check
                </label>
                {form.configureHealth ? (
                  <div className="form-fields-grid">
                    <label>
                      Health name
                      <input
                        value={form.healthName}
                        onChange={(event) => {
                          setValidationMessage("");
                          setForm((current) => ({
                            ...current,
                            healthName: event.target.value,
                          }));
                        }}
                      />
                    </label>
                    <label>
                      Type
                      <select
                        value={form.healthType}
                        onChange={(event) => {
                          setValidationMessage("");
                          setForm((current) => ({
                            ...current,
                            healthType: event.target.value,
                          }));
                        }}
                      >
                        <option value="HEALTH_CHECK_TYPE_HTTP">HTTP</option>
                        <option value="HEALTH_CHECK_TYPE_HTTPS">HTTPS</option>
                        <option value="HEALTH_CHECK_TYPE_GRPC">gRPC</option>
                        <option value="HEALTH_CHECK_TYPE_TCP">TCP</option>
                        <option value="HEALTH_CHECK_TYPE_UDP">UDP</option>
                      </select>
                    </label>
                    <label>
                      Health path
                      <input
                        value={form.healthPath}
                        onChange={(event) => {
                          setValidationMessage("");
                          setForm((current) => ({
                            ...current,
                            healthPath: event.target.value,
                          }));
                        }}
                      />
                    </label>
                    <label>
                      Interval seconds
                      <input
                        min="1"
                        type="number"
                        value={form.healthIntervalSeconds}
                        onChange={(event) => {
                          setValidationMessage("");
                          setForm((current) => ({
                            ...current,
                            healthIntervalSeconds: Number(event.target.value),
                          }));
                        }}
                      />
                    </label>
                    <label>
                      Timeout seconds
                      <input
                        min="1"
                        type="number"
                        value={form.healthTimeoutSeconds}
                        onChange={(event) => {
                          setValidationMessage("");
                          setForm((current) => ({
                            ...current,
                            healthTimeoutSeconds: Number(event.target.value),
                          }));
                        }}
                      />
                    </label>
                  </div>
                ) : null}
              </div>
              {form.configureHealth && !readiness[3] ? (
                <p className="step-note">
                  Health checks need a name, type, interval, and timeout.
                </p>
              ) : null}
              <div className="wizard-review">
                <span>Review</span>
                <dl>
                  <dt>Environment</dt>
                  <dd>
                    {selectedEnvironment?.name ?? "No environment selected"}
                  </dd>
                  <dt>Instance</dt>
                  <dd>
                    {form.instanceName || "Unnamed"} at{" "}
                    {form.address || "no address"}
                  </dd>
                  <dt>Primary endpoint</dt>
                  <dd>
                    {summaryEndpoint
                      ? `${formatProtocol(summaryEndpoint.protocol).toUpperCase()} :${summaryEndpoint.port}${summaryEndpoint.path || ""}`
                      : "No primary endpoint"}
                  </dd>
                  <dt>Health</dt>
                  <dd>
                    {form.configureHealth
                      ? `${form.healthName} every ${form.healthIntervalSeconds}s`
                      : "Not configured"}
                  </dd>
                </dl>
              </div>
            </div>
          ) : null}
        </div>

        {registrationSuccess ? (
          <p className="success">{registrationSuccess}</p>
        ) : null}
        {validationMessage ? (
          <p className="wizard-feedback" role="status">
            {validationMessage}
          </p>
        ) : null}
        <div className="wizard-actions">
          <button
            className="button-ghost"
            type="button"
            onClick={() => {
              setValidationMessage("");
              onStepChange(0);
            }}
          >
            Cancel
          </button>
          <div className="wizard-action-group">
            {step > 0 ? (
              <button className="button-secondary" type="button" onClick={onBack}>
                Back
              </button>
            ) : null}
            {step < steps.length - 1 ? (
              <button type="submit">
                {canContinue ? "Continue" : "Show what is missing"}
              </button>
            ) : (
              <button disabled={saving || environments.length === 0} type="submit">
                {saving ? "Registering" : "Register runtime"}
              </button>
            )}
          </div>
        </div>
      </div>
    </form>
  );
}

function WizardStepper({
  currentStep,
  readiness,
  onStepClick,
}: {
  currentStep: number;
  readiness: boolean[];
  onStepClick: (step: number) => void;
}) {
  return (
    <ol className="wizard-stepper" aria-label="Registration steps">
      {steps.map((item, index) => {
        const completed = index < currentStep && readiness[index];
        const current = index === currentStep;
        return (
          <li
            className={
              completed
                ? "complete"
                : current
                  ? "current"
                  : "upcoming"
            }
            key={item.id}
          >
            <button
              aria-current={current ? "step" : undefined}
              onClick={() => onStepClick(index)}
              type="button"
            >
              <span className="wizard-step-marker" aria-hidden="true">
                {completed ? "✓" : index + 1}
              </span>
              <span>{item.label}</span>
            </button>
            {index < steps.length - 1 ? <span className="wizard-connector" /> : null}
          </li>
        );
      })}
    </ol>
  );
}

export function getRegistrationReadiness(
  form: RegistrationFormState,
  hasSelectedService: boolean,
) {
  const targetReady = Boolean(hasSelectedService && form.environmentId);
  const instanceReady = Boolean(
    form.instanceName.trim() && form.address.trim(),
  );
  const endpointsReady =
    form.endpoints.length > 0 &&
    form.endpoints.every(
      (endpoint) => endpoint.name.trim() && endpoint.port > 0,
    ) &&
    form.endpoints.some((endpoint) => endpoint.primary);
  const healthReady =
    !form.configureHealth ||
    Boolean(
      form.healthName.trim() &&
        form.healthType &&
        form.healthIntervalSeconds > 0 &&
        form.healthTimeoutSeconds > 0,
    );

  return [targetReady, instanceReady, endpointsReady, healthReady];
}

const stepHelp = [
  "Choose a service and environment before moving on.",
  "Add an instance name and reachable address.",
  "Complete at least one endpoint with a name, port, and primary selection.",
  "Complete the health check fields or turn health check off.",
];

function formatProtocol(value: string) {
  return value.replace("PROTOCOL_", "").toLowerCase();
}
