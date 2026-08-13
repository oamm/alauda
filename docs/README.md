# Service Registry Architecture & Design Documentation

Welcome to the complete architectural specification for the Service Registry MVP.

This repository contains the complete blueprint for a modern, lightweight, self-hosted service registry and service health administration platform.

---

## 📚 Documentation Structure

The following documents provide the complete architectural and implementation blueprint:

### Core Architecture

**[ARCHITECTURE.md](ARCHITECTURE.md)** - The master architectural document

- Overview of the system's six primary responsibilities
- High-level component architecture and data flow patterns
- 19 key architectural decisions with rationale
- Concurrency model and worker pool design
- Database design philosophy
- API design principles
- Security model
- Future extension points

**Recommended first read**: Start here to understand the big picture.

---

### Domain Model

**[DOMAIN.md](DOMAIN.md)** - Complete domain entity definitions

- 15 core entities (Service, Deployment, Instance, Endpoint, HealthCheck, HealthState, HealthResult, Incident, AlertPolicy, NotificationChannel, Event, AuditLog, User, ApiToken, etc.)
- Complete property definitions for each entity
- Constraints and invariants
- Relationships and aggregates
- Value objects and enumerations
- Domain events

**When needed**: Refer to this when implementing entities or understanding relationships.

---

### Database Design

**[DATABASE.md](DATABASE.md)** - Complete SQLite schema specification

- 17 normalized tables with all columns, types, and constraints
- Foreign key relationships and cascade rules
- Indexes optimized for query patterns
- Index justification and performance targets
- Migration strategy
- Data cleanup and retention policies
- Backup strategy
- Growth planning and capacity estimates
- WAL and transaction model

**When needed**: Reference this for SQL implementations and repository code.

---

### API Specification

**[API.md](API.md)** - Complete Protobuf API contracts

- 10 ConnectRPC services with all RPCs defined
- Message definitions for all request/response types
- Enumerations and common types
- Pagination, streaming, and error handling patterns
- Service lookup API (discovery)
- API design principles
- buf.yaml and buf.gen.yaml configurations

**When needed**: Use this to implement API handlers and generate clients.

---

### Health Check Scheduler

**[HEALTH.md](HEALTH.md)** - Deep dive into health monitoring design

- Complete scheduler architecture with detailed diagrams
- Scheduler loop, worker pool, and state transition engine
- Check queue and result queue design
- Event publishing and alert triggering
- Execution patterns for HTTP, TCP, gRPC, and heartbeat checks
- Lifecycle management and graceful shutdown
- Metrics and observability
- Failure scenarios and recovery
- Configuration options
- Testing strategy

**When needed**: Essential for understanding and implementing the health subsystem.

---

### MVP Implementation Plan

**[MVP.md](MVP.md)** - Phase-by-phase implementation roadmap

- Repository structure and package layout
- Dependency flow and architectural boundaries
- 6 implementation phases (Foundation → Security & Hardening)
- Detailed tasks for each phase with deliverables
- Testing strategy and code quality guidelines
- Success criteria for each phase
- Risk mitigation strategies

**When needed**: Use this to plan sprints and track progress.

---

### Architectural Risks

**[RISKS.md](RISKS.md)** - Risk analysis and mitigation strategies

- 13 critical and operational risks identified
- Risk level, symptoms, and mitigation for each
- Monitoring strategy and alert rules
- Risk matrix prioritizing by impact
- Specific code examples showing good vs. bad patterns
- Configuration best practices

**When needed**: Review before implementing high-risk components.

**[CONFIGURATION.md](CONFIGURATION.md)** - Runtime configuration reference.

**[DEPLOYMENT.md](DEPLOYMENT.md)** - Production startup, backups, retention, and monitoring.

**[CLI.md](CLI.md)** - Command-line usage, auth, audit, and output formats.

---

## 🏗️ Architecture Overview

### System Responsibilities

```
Service Registry
│
├── Service Catalog          (CRUD: services, deployments, instances, endpoints)
├── Health Monitoring        (Scheduled health checks via worker pool)
├── Health State Management  (Deterministic state transitions)
├── Incident Tracking        (Outage lifecycle: open → resolved)
├── Alert Engine             (Webhook, email notifications with cooldown)
└── Administration UI        (Modern web dashboard + API)
```

### Key Architectural Decisions

1. **Environment is First-Class** - Not just a tag; primary domain concept
2. **Health Checks Execute Outside Transactions** - No database locks during network I/O
3. **Bounded Worker Pool** - Fixed concurrency, no unbounded goroutines
4. **Deterministic State Transitions** - Explicit rules, not side effects
5. **Monorepo with Clear Boundaries** - Backend (Go), Frontend (React), API (Protobuf)
6. **Repository Pattern** - Decouple domain from persistence
7. **Domain Events** - Decoupled event publishing for audit and alerts
8. **Soft Delete** - Preserve historical data
9. **Simple Alert System** - Webhook-first enables unlimited integrations
10. **SQLite → PostgreSQL Ready** - Repository interface allows backend swap

---

## 📋 Document Quick Reference

### By Role

**Software Architect**:

- Read: ARCHITECTURE.md, DOMAIN.md, HEALTH.md
- Reference: API.md, DATABASE.md, RISKS.md

**Backend Engineer**:

- Read: DOMAIN.md, API.md, HEALTH.md, DATABASE.md
- Reference: ARCHITECTURE.md, MVP.md

**Frontend Engineer**:

- Read: ARCHITECTURE.md, API.md, MVP.md
- Reference: DOMAIN.md

**DevOps/Platform**:

- Read: ARCHITECTURE.md, MVP.md
- Reference: RISKS.md, DATABASE.md

**Tech Lead**:

- Read: All documents (in order)
- Special attention: ARCHITECTURE.md, HEALTH.md, RISKS.md, MVP.md

---

### By Task

**Implementing a new domain entity**:

1. Check DOMAIN.md for entity definition
2. Check DATABASE.md for schema
3. Check ARCHITECTURE.md for constraints

**Implementing an API endpoint**:

1. Check API.md for RPC contract
2. Check ARCHITECTURE.md for architecture pattern
3. Use DOMAIN.md for entity details

**Setting up the health scheduler**:

1. Read HEALTH.md completely
2. Reference ARCHITECTURE.md for concurrency model
3. Check RISKS.md for failure scenarios

**Configuring for production**:

1. Read MVP.md Phase 6 (Security & Hardening)
2. Review RISKS.md for monitoring alerts
3. Check DATABASE.md for retention policies

---

## 🎯 Quick Start for Development

### Prerequisites

- Go 1.21+
- Node.js 20+
- SQLite 3.40+
- Buf (for Protobuf code generation)

### Initial Setup

1. **Read the architecture documents** (ARCHITECTURE.md first)
2. **Understand the domain model** (DOMAIN.md)
3. **Review the implementation plan** (MVP.md Phase 0)
4. **Set up development environment**:

   ```bash
   git clone https://github.com/company/service-registry.git
   cd service-registry
   make setup      # Install dependencies
   make dev        # Start development server
   ```

5. **Run initial tests**:

   ```bash
   go test ./...
   cd web && npm test
   ```

6. **Run scoped Go coverage on Windows PowerShell**:
   ```powershell
   powershell -ExecutionPolicy Bypass -File scripts\coverage.ps1
   ```
   The coverage gate excludes generated protobuf, embedded web static output, frontend dependencies, and server entrypoint packages. It defaults to an 80% threshold; use `-Threshold 0` to generate a report without failing while coverage is still being raised.

### Development Workflow

1. Pick a task from MVP.md
2. Check relevant architecture documents
3. Implement with tests
4. Verify against RISKS.md
5. Create PR

---

## 🏁 Implementation Phases

### Phase 0: Foundation (Week 1)

**Status**: Not started
**Goal**: Server running, database initialized, API working

- Go server scaffolding
- SQLite with migrations
- Protobuf code generation
- React UI framework
- Docker build

### Phase 1: Service Catalog (Week 2)

**Status**: Not started
**Goal**: Core registry functionality

- CRUD for services, deployments, instances, endpoints
- CLI tools
- Basic UI

### Phase 2: Health Checks (Week 3)

**Status**: Not started
**Goal**: Health monitoring

- Health check scheduling
- Worker pool
- HTTP/TCP/gRPC checks
- State transitions

### Phase 3: Incidents & Availability (Week 4)

**Status**: Not started
**Goal**: Outage tracking

- Incident creation/resolution
- Availability calculation
- Event streaming

### Phase 4: Alerts (Week 5)

**Status**: Not started
**Goal**: Notifications

- Alert policies
- Webhook channel
- Email channel

### Phase 5: Operational UI (Week 6)

**Status**: Not started
**Goal**: Complete dashboard

- Dashboard
- All pages and views
- Real-time updates

### Phase 6: Security & Hardening (Week 7)

**Status**: Mostly complete; environment-level permissions remain future scope
**Goal**: Production ready

Remaining MVP hardening work is tracked in `MVP.md`. The scoped `>80%` Go coverage gate and ConnectRPC `TestNotificationChannel` proto/handler are complete; environment-level permissions remain future scope.

- Authentication and authorization
- Audit logging
- Observability
- Production configuration

---

## 📊 Key Metrics & Targets

### Performance Targets

- List services by environment: **< 100ms** (1000s of services)
- List instances by deployment: **< 50ms** (100s of instances)
- Health check execution: **< 50ms p99** (1000+ concurrent)
- Health state persistence: **< 10ms**
- Dashboard load: **< 500ms**

### Scale Targets (MVP)

- 1,000 services ✓
- 10,000 service instances ✓
- 20,000 endpoints ✓
- 20,000 health checks ✓
- 100 health check results per second ✓

### Availability

- API: **99.9%** uptime
- Health monitoring: Continuous (automatic recovery on restart)
- Alerts: Best-effort (retry up to 24 hours)

---

## 🔐 Security Model

### Authentication (MVP)

- Local user accounts (username/password)
- API tokens for automation
- Session-based for web UI

### Authorization

- RBAC with roles: Administrator, Operator, Viewer, Automation
- Environment-level permissions (future)

### Audit

- All mutations logged
- Changes tracked (before/after values)
- 100% retention indefinite

### Future Enhancements

- OIDC integration
- SSO support
- Service-level permissions
- Fine-grained RBAC

---

## 🚀 Production Deployment

### Recommended Setup

```
Registry Server (single instance for MVP)
  ├─ Embedded React SPA
  ├─ ConnectRPC API
  ├─ Health Scheduler
  ├─ Alert Engine
  └─ SQLite Database (with WAL)

Configuration
  ├─ YAML file
  ├─ Environment variables
  └─ CLI flags

Monitoring (external)
  ├─ Prometheus for metrics
  ├─ Loki for logs
  └─ Grafana for dashboards

Notifications (external)
  ├─ Slack webhook
  ├─ Email (SMTP)
  └─ Custom (webhook)
```

### Deployment Checklist

- [ ] Read MVP.md Phase 6 (Security & Hardening)
- [ ] Review RISKS.md (monitoring alerts)
- [ ] Configure YAML (production settings)
- [ ] Set up backups (daily SQLite backups)
- [ ] Enable monitoring (Prometheus scrape)
- [ ] Configure notifications (webhook/email)
- [ ] Create runbooks (incident response)
- [ ] Test disaster recovery (restore from backup)

---

## 📖 Reading Order

### For Project Stakeholders

1. This README (introduction)
2. ARCHITECTURE.md (overview)
3. MVP.md (implementation plan)

### For Architects

1. ARCHITECTURE.md (complete architecture)
2. DOMAIN.md (entities and relationships)
3. HEALTH.md (complex subsystem)
4. RISKS.md (failure modes and mitigation)
5. DATABASE.md (schema and indexing)
6. API.md (API contracts)
7. MVP.md (implementation roadmap)

### For Backend Engineers

1. ARCHITECTURE.md (understand the design)
2. DOMAIN.md (entity definitions)
3. MVP.md (Phase 0 tasks)
4. DATABASE.md (schema)
5. API.md (RPC contracts)
6. HEALTH.md (scheduler details)

### For Frontend Engineers

1. ARCHITECTURE.md (system overview)
2. API.md (API contracts)
3. MVP.md Phase 0 (setup React)
4. MVP.md Phase 1-5 (page implementations)

### For DevOps

1. ARCHITECTURE.md (system design)
2. MVP.md Phase 0 & 6 (deployment)
3. RISKS.md (monitoring and alerts)
4. DATABASE.md (backups and scale)

---

## ❓ FAQ

**Q: Why Protobuf instead of REST/JSON?**
A: Type-safe API contracts, auto-generated clients, efficient serialization, and support for streaming. Protobuf + ConnectRPC gives us gRPC benefits with HTTP/1.1 compatibility.

**Q: Why not use Kubernetes for distributed registry?**
A: MVP targets single instance simplicity. Kubernetes adds operational complexity. PostgreSQL provides horizontal scaling without distributed consensus.

**Q: Why not implement a full metrics/monitoring platform?**
A: Out of scope. The registry is a service registry + health monitor, not a metrics platform. OpenTelemetry integration allows bringing your own monitoring backend.

**Q: Can we add _____ feature?**
A: Check the "Explicitly Out of Scope" section in ARCHITECTURE.md. New features must serve the core mission: "Service Registry + Health Monitoring + Registry Alerts + Administration."

**Q: How do we scale beyond 1 registry instance?**
A: Phase for MVP is single instance. Future designs in ARCHITECTURE.md section 14 (Future Extension Points) include:

- PostgreSQL backend (replaces SQLite)
- Shared database with multiple instances
- Distributed event bus for multi-instance coordination
- High availability setup with leader election

**Q: Can we add service mesh integration?**
A: No. Service mesh is out of scope. The registry provides information about service locations; it doesn't provide routing, load balancing, or traffic management.

---

## 🤝 Contributing

When implementing features:

1. Read the relevant architecture document
2. Review RISKS.md for similar features
3. Write tests alongside code
4. Update documentation
5. Ensure monitoring in place

---

## 📞 Architecture Review Checklist

Before implementing a significant feature:

- [ ] Reviewed ARCHITECTURE.md
- [ ] Checked DOMAIN.md for relevant entities
- [ ] Verified DATABASE.md schema supports use case
- [ ] Designed API in PROTO format (matches API.md patterns)
- [ ] Reviewed RISKS.md for similar patterns
- [ ] Planned monitoring (metrics + alerts)
- [ ] Identified failure scenarios
- [ ] Documented in code comments
- [ ] Created tests before implementation

---

## 📝 Version History

| Version | Date       | Status              | Notes                                                                             |
| ------- | ---------- | ------------------- | --------------------------------------------------------------------------------- |
| 1.0     | 2026-08-12 | **DESIGN COMPLETE** | Architecture and all design documents complete. Ready for Phase 0 implementation. |

---

## 📚 Additional Resources

### External References

- [gRPC HTTP/1.1](https://connectrpc.com/)
- [Protocol Buffers](https://developers.google.com/protocol-buffers)
- [Buf](https://buf.build/)
- [SQLite Best Practices](https://www.sqlite.org/bestpractice.html)
- [Go Concurrency Patterns](https://go.dev/blog/pipelines)

### Recommended Reading

- "Building Microservices" by Sam Newman (architecture patterns)
- "Release It!" by Michael Nygard (operational concerns)
- "The Twelve-Factor App" (cloud-native principles)

---

## 🎓 Next Steps

1. **Review**: Read ARCHITECTURE.md completely
2. **Discuss**: Schedule design review with team
3. **Plan**: Break down Phase 0 into sprint tasks
4. **Implement**: Begin Phase 0 foundation work
5. **Iterate**: Gather feedback, adjust as needed

The architecture is complete and validated. The foundation is solid for building a production-grade service registry.

**Let's build something great!** 🚀
