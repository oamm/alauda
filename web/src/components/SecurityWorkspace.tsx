import { useEffect, useState, type FormEvent, type ReactNode } from "react";
import { Copy, Plus, RefreshCw } from "lucide-react";

import {
  type ApiToken,
  type ApplicationKey,
  type Environment,
  type Session,
  type UserAccount,
  createApiToken,
  createApplicationKey,
  createUser,
  credentialCapabilities,
  listApiTokens,
  listApplicationKeys,
  listSessions,
  listUsers,
  revokeApiToken,
  revokeApplicationKey,
  revokeSession,
} from "../api";
import { formatTimestamp } from "../utils/format";
import {
  ActionMenu,
  Alert,
  Button,
  Checkbox,
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  EmptyState,
  FilterBar,
  FormField,
  FormSection,
  Input,
  PageHeader,
  Pagination,
  PasswordInput,
  ResourceList,
  ResourceRow,
  SearchInput,
  SectionHeader,
  Select,
  Skeleton,
  StatusBadge,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
  Workspace,
} from "./ui";

type SecurityTab = "users" | "tokens" | "keys" | "sessions";
type Resource = {
  id: string;
  cells: ReactNode[];
  title: ReactNode;
  description: ReactNode;
  status: ReactNode;
  action?: ReactNode;
};
const pageSize = 25;
const tabletOptionalColumns = new Set(["Email", "Created", "Last used", "Expires", "IP"]);
const initialUser = { username: "", displayName: "", email: "", password: "", role: "Viewer" };
const initialToken = { name: "", scopes: ["registry.read"], expiresInHours: "720" };
const initialKey = { name: "", scopes: ["discovery.read"], environmentIds: [] as string[], expiresInHours: "720" };

function errorMessage(error: unknown, fallback: string) {
  return error instanceof Error ? error.message : fallback;
}

function expiration(hours: string) {
  const value = Number(hours);
  return Number.isFinite(value) && value > 0
    ? new Date(Date.now() + value * 60 * 60 * 1000).toISOString()
    : undefined;
}

function statusFor(enabled: boolean, expiresAt?: string, revokedAt?: string) {
  if (revokedAt || !enabled) return <StatusBadge status="revoked" />;
  if (expiresAt && Date.parse(expiresAt) <= Date.now())
    return <StatusBadge status="expired" />;
  return <StatusBadge status="active" />;
}

function ScopeChoices({ value, onChange }: { value: string[]; onChange: (value: string[]) => void }) {
  return (
    <FormSection title="Scopes" description="Grant only the permissions this credential needs.">
      <div className="grid gap-2 sm:grid-cols-2">
        {credentialCapabilities.map((scope) => (
          <label className="flex min-h-9 items-center gap-2 text-sm" key={scope}>
            <Checkbox
              checked={value.includes(scope)}
              aria-label={scope}
              onCheckedChange={(checked) =>
                onChange(checked === true ? [...value, scope] : value.filter((item) => item !== scope))
              }
            />
            <span>{scope}</span>
          </label>
        ))}
      </div>
    </FormSection>
  );
}

function ResourceView({
  title, description, columns, rows, emptyTitle, emptyDescription, action,
  search, onSearchChange, filter, filtered, loading,
}: {
  title: string;
  description: string;
  columns: string[];
  rows: Resource[];
  emptyTitle: string;
  emptyDescription: string;
  action?: ReactNode;
  search: string;
  onSearchChange: (value: string) => void;
  filter?: ReactNode;
  filtered: boolean;
  loading?: boolean;
}) {
  const [page, setPage] = useState(1);
  const count = Math.max(1, Math.ceil(rows.length / pageSize));
  const current = Math.min(page, count);
  const visible = rows.slice((current - 1) * pageSize, current * pageSize);
  return (
    <Workspace className="min-w-0 !p-0">
      <SectionHeader title={title} description={description} action={action} />
      <FilterBar compact>
        <FormField label="Search">
          <SearchInput value={search} onChange={(event) => { onSearchChange(event.target.value); setPage(1); }} placeholder={`Search ${title.toLowerCase()}`} />
        </FormField>
        {filter}
      </FilterBar>
      {loading ? (
        <div className="grid gap-2 p-4" aria-label={`Loading ${title.toLowerCase()}`}>
          {[0, 1, 2].map((index) => <Skeleton className="h-9 w-full" key={index} />)}
        </div>
      ) : rows.length ? (
        <>
          <div className="hidden md:block">
            <Table aria-label={title}>
              <TableHeader><TableRow>{columns.map((column) => <TableHead className={tabletOptionalColumns.has(column) ? "hidden lg:table-cell" : undefined} key={column}>{column}</TableHead>)}</TableRow></TableHeader>
              <TableBody>{visible.map((row) => (
                <TableRow key={row.id}>{row.cells.map((cell, index) =>
                  <TableCell className={tabletOptionalColumns.has(columns[index]) ? "hidden lg:table-cell" : undefined} key={columns[index]}>{cell}</TableCell>)}</TableRow>
              ))}</TableBody>
            </Table>
          </div>
          <div className="md:hidden">
            <ResourceList label={title}>{visible.map((row) => (
              <ResourceRow key={row.id} title={row.title} description={row.description} status={row.status} action={row.action} />
            ))}</ResourceList>
          </div>
          {rows.length > pageSize ? <Pagination page={current} pageCount={count} onPageChange={setPage} /> : null}
        </>
      ) : <EmptyState
        title={filtered ? `No matching ${title.toLowerCase()}` : emptyTitle}
        description={filtered ? "Try a different search or status." : emptyDescription}
      />}
    </Workspace>
  );
}

export function SecurityWorkspace({ environments }: { environments: Environment[] }) {
  const [tab, setTab] = useState<SecurityTab>("users");
  const [users, setUsers] = useState<UserAccount[]>([]);
  const [tokens, setTokens] = useState<ApiToken[]>([]);
  const [keys, setKeys] = useState<ApplicationKey[]>([]);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [ownerId, setOwnerId] = useState("");
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [dialog, setDialog] = useState<"user" | "token" | "key" | "revoke" | null>(null);
  const [revokeTarget, setRevokeTarget] = useState<{ id: string; kind: "session" | "token" | "key"; name: string }>();
  const [userForm, setUserForm] = useState(initialUser);
  const [tokenForm, setTokenForm] = useState(initialToken);
  const [keyForm, setKeyForm] = useState(initialKey);
  const [secret, setSecret] = useState("");
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(true);
  const [tokensLoading, setTokensLoading] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  async function refresh() {
    setLoading(true);
    try {
      const [nextUsers, nextKeys, nextSessions] = await Promise.all([
        listUsers(), listApplicationKeys(), listSessions(),
      ]);
      setUsers(nextUsers);
      setKeys(nextKeys);
      setSessions(nextSessions);
      setOwnerId((current) => current || nextUsers[0]?.id || "");
    } finally {
      setLoading(false);
    }
  }
  useEffect(() => {
    refresh().catch((err: unknown) => setError(errorMessage(err, "Failed to load Security data")));
  }, []);
  useEffect(() => {
    if (!ownerId) { setTokens([]); setTokensLoading(false); return; }
    let active = true;
    setTokensLoading(true);
    listApiTokens(ownerId).then((items) => { if (active) setTokens(items); }).catch((err: unknown) => {
      if (active) setError(errorMessage(err, "Failed to load API tokens"));
    }).finally(() => { if (active) setTokensLoading(false); });
    return () => { active = false; };
  }, [ownerId]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      if (dialog === "user") {
        const created = await createUser(userForm);
        setUserForm(initialUser);
        setDialog(null);
        setMessage(`Created user ${created.username}.`);
        await refresh();
      } else if (dialog === "token") {
        const created = await createApiToken({
          userId: ownerId, name: tokenForm.name, scopes: tokenForm.scopes,
          expiresAt: expiration(tokenForm.expiresInHours),
        });
        setTokenForm(initialToken);
        setSecret(created.secret || "");
        if (!created.secret) setDialog(null);
        setMessage("API token created.");
        setTokens(await listApiTokens(ownerId));
      } else if (dialog === "key") {
        const created = await createApplicationKey({
          name: keyForm.name, scopes: keyForm.scopes,
          environmentIds: keyForm.environmentIds,
          expiresAt: expiration(keyForm.expiresInHours),
        });
        setKeyForm(initialKey);
        setSecret(created.secret || "");
        if (!created.secret) setDialog(null);
        setMessage("Application key created.");
        setKeys(await listApplicationKeys());
      }
    } catch (err) {
      setError(errorMessage(err, "Unable to create credential"));
    } finally {
      setBusy(false);
    }
  }

  async function confirmRevoke() {
    if (!revokeTarget) return;
    setBusy(true);
    setError("");
    try {
      if (revokeTarget.kind === "session") {
        await revokeSession(revokeTarget.id);
        setSessions(await listSessions());
      } else if (revokeTarget.kind === "token") {
        await revokeApiToken(revokeTarget.id);
        setTokens(await listApiTokens(ownerId));
      } else {
        await revokeApplicationKey(revokeTarget.id);
        setKeys(await listApplicationKeys());
      }
      setDialog(null);
      setRevokeTarget(undefined);
      setMessage(`${revokeTarget.name} revoked.`);
    } catch (err) {
      setError(errorMessage(err, "Failed to revoke access"));
    } finally {
      setBusy(false);
    }
  }

  function openRevoke(id: string, kind: "session" | "token" | "key", name: string) {
    setRevokeTarget({ id, kind, name });
    setDialog("revoke");
    setError("");
  }
  function closeDialog() {
    setDialog(null);
    setRevokeTarget(undefined);
    setSecret("");
    setError("");
  }
  function changeTab(next: string) {
    setTab(next as SecurityTab);
    setSearch("");
    setStatusFilter("all");
    setMessage("");
  }
  const matches = (parts: Array<string | undefined>) =>
    parts.join(" ").toLowerCase().includes(search.trim().toLowerCase());
  const statusMatches = (enabled: boolean, expiresAt?: string, revokedAt?: string) => {
    if (statusFilter === "all") return true;
    const state = revokedAt || !enabled ? "revoked" : expiresAt && Date.parse(expiresAt) <= Date.now() ? "expired" : "active";
    return state === statusFilter;
  };
  const statusFilterControl = (
    <FormField label="Status">
      <Select value={statusFilter} onChange={(event) => setStatusFilter(event.target.value)}>
        <option value="all">All statuses</option>
        <option value="active">Active</option>
        {tab === "users" ? <option value="disabled">Disabled</option> : (
          <>
            <option value="expired">Expired</option>
            <option value="revoked">Revoked</option>
          </>
        )}
      </Select>
    </FormField>
  );
  const filtered = Boolean(search.trim()) || statusFilter !== "all";
  const userRows: Resource[] = users.filter((user) => matches([user.username, user.displayName, user.email, user.role]) &&
    (statusFilter === "all" || (user.enabled ? "active" : "disabled") === statusFilter)).map((user) => {
    const status = <StatusBadge status={user.enabled ? "active" : "disabled"} />;
    return { id: user.id, title: <strong>{user.username}</strong>,
      description: <span>{user.displayName || user.email} · {user.role}</span>, status,
      cells: [<strong>{user.username}</strong>, user.displayName || "", user.email, user.role,
        user.createdAt ? formatTimestamp(user.createdAt) : "", status, null] };
  });
  const tokenRows: Resource[] = tokens.filter((token) =>
    matches([token.name, token.scopes.join(" "), users.find((user) => user.id === token.userId)?.username]) &&
    statusMatches(token.enabled, token.expiresAt)).map((token) => {
    const status = statusFor(token.enabled, token.expiresAt);
    const action = token.enabled ? <ActionMenu label={`Actions for ${token.name}`} items={[
      { label: "Revoke", onSelect: () => openRevoke(token.id, "token", token.name) },
    ]} /> : null;
    return { id: token.id, title: <strong>{token.name}</strong>,
      description: <span>{users.find((user) => user.id === token.userId)?.username || token.userId} · {token.scopes.join(", ")}</span>,
      status, action,
      cells: [<strong>{token.name}</strong>, users.find((user) => user.id === token.userId)?.username || token.userId,
        token.scopes.join(", "), token.createdAt ? formatTimestamp(token.createdAt) : "",
        token.lastUsedAt ? formatTimestamp(token.lastUsedAt) : "Never",
        token.expiresAt ? formatTimestamp(token.expiresAt) : "Never", status, action] };
  });
  const keyRows: Resource[] = keys.filter((key) =>
    matches([key.name, key.scopes.join(" "), ...(key.environmentIds || []).map((id) => environments.find((env) => env.id === id)?.key)]) &&
    statusMatches(key.enabled, key.expiresAt)).map((key) => {
    const status = statusFor(key.enabled, key.expiresAt);
    const action = key.enabled ? <ActionMenu label={`Actions for ${key.name}`} items={[
      { label: "Revoke", onSelect: () => openRevoke(key.id, "key", key.name) },
    ]} /> : null;
    const access = key.environmentIds?.length ? key.environmentIds.map((id) =>
      environments.find((env) => env.id === id)?.key || id).join(", ") : "All environments";
    return { id: key.id, title: <strong>{key.name}</strong>,
      description: <span>{key.scopes.join(", ")} · {access}</span>, status, action,
      cells: [<strong>{key.name}</strong>, key.scopes.join(", "), access,
        key.createdAt ? formatTimestamp(key.createdAt) : "",
        key.expiresAt ? formatTimestamp(key.expiresAt) : "Never", status, action] };
  });
  const sessionRows: Resource[] = sessions.filter((session) =>
    matches([session.userId, session.userAgent, session.clientIp]) &&
    statusMatches(!session.revokedAt, session.expiresAt, session.revokedAt)).map((session) => {
    const user = users.find((item) => item.id === session.userId)?.username || session.userId;
    const status = statusFor(!session.revokedAt, session.expiresAt, session.revokedAt);
    const action = !session.revokedAt ? <ActionMenu label={`Actions for ${user} session`} items={[
      { label: "Revoke", onSelect: () => openRevoke(session.id, "session", `${user} session`) },
    ]} /> : null;
    return { id: session.id, title: <strong>{user}</strong>,
      description: <span>{session.clientIp || "IP unknown"} · {session.userAgent || "Unknown client"}</span>, status, action,
      cells: [<strong>{user}</strong>, session.userAgent || "Unknown client",
        session.clientIp || "", formatTimestamp(session.lastActivityAt),
        formatTimestamp(session.createdAt), formatTimestamp(session.expiresAt), status, action] };
  });

  return (
    <section className="grid min-w-0 gap-3">
      <PageHeader title="Security" description="Manage users, credentials, and active sessions." />
      {error && !dialog ? <Alert tone="danger" title="Security data unavailable">{error}</Alert> : null}
      {message && !dialog ? <Alert tone="success" title={message} /> : null}
      <Tabs value={tab} onValueChange={changeTab} className="min-w-0">
        <TabsList aria-label="Security views">
          <TabsTrigger value="users">Users</TabsTrigger>
          <TabsTrigger value="tokens">API tokens</TabsTrigger>
          <TabsTrigger value="keys">Application keys</TabsTrigger>
          <TabsTrigger value="sessions">Sessions</TabsTrigger>
        </TabsList>
        <TabsContent value="users">
          <ResourceView title="Users" description="Manage people who can access Alauda."
            columns={["Username", "Name", "Email", "Role", "Created", "Status", "Actions"]}
            rows={userRows} search={search} onSearchChange={setSearch} filter={statusFilterControl} filtered={filtered} loading={loading}
            emptyTitle="No users" emptyDescription="Create a user to grant access."
            action={<Button variant="primary" onClick={() => { setError(""); setDialog("user"); }}><Plus size={16} />Create user</Button>} />
        </TabsContent>
        <TabsContent value="tokens">
          <ResourceView title="API tokens" description="Personal credentials used to access Alauda APIs."
            columns={["Name", "Owner", "Scopes", "Created", "Last used", "Expires", "Status", "Actions"]}
            rows={tokenRows} search={search} onSearchChange={setSearch} filtered={filtered} loading={loading || tokensLoading}
            filter={<><FormField label="Owner"><Select value={ownerId} onChange={(event) => setOwnerId(event.target.value)}>
              {users.map((user) => <option key={user.id} value={user.id}>{user.username}</option>)}
            </Select></FormField>{statusFilterControl}</>}
            emptyTitle="No API tokens" emptyDescription="Create a token for this user to access Alauda APIs."
            action={<Button variant="primary" disabled={!ownerId} onClick={() => { setError(""); setDialog("token"); }}><Plus size={16} />Create API token</Button>} />
        </TabsContent>
        <TabsContent value="keys">
          <ResourceView title="Application keys" description="Credentials used by services and automation."
            columns={["Name", "Scopes", "Environment access", "Created", "Expires", "Status", "Actions"]}
            rows={keyRows} search={search} onSearchChange={setSearch} filter={statusFilterControl} filtered={filtered} loading={loading}
            emptyTitle="No application keys" emptyDescription="Create a key for service-to-service access or automation."
            action={<Button variant="primary" onClick={() => { setError(""); setDialog("key"); }}><Plus size={16} />Create application key</Button>} />
        </TabsContent>
        <TabsContent value="sessions">
          <ResourceView title="Sessions" description="Review active sessions and revoke access you do not recognize."
            columns={["User", "Client", "IP", "Last active", "Created", "Expires", "Status", "Actions"]}
            rows={sessionRows} search={search} onSearchChange={setSearch} filter={statusFilterControl} filtered={filtered} loading={loading}
            emptyTitle="No sessions" emptyDescription="Authenticated sessions will appear here."
            action={<Button onClick={() => refresh().catch((err: unknown) => setError(errorMessage(err, "Failed to refresh sessions")))}><RefreshCw size={16} />Refresh</Button>} />
        </TabsContent>
      </Tabs>
      <Dialog open={dialog !== null} onOpenChange={(open) => { if (!open && !busy) closeDialog(); }}>
        <DialogContent size={dialog === "key" ? "lg" : "md"}>
          <DialogHeader>
            <DialogTitle>{secret ? `${dialog === "token" ? "API token" : "Application key"} created` :
              dialog === "revoke" ? "Revoke access" : dialog === "user" ? "Create user" :
              dialog === "token" ? "Create API token" : "Create application key"}</DialogTitle>
            <DialogDescription>
              {secret ? "Copy this value now. It will not be shown again." :
                dialog === "revoke" ? `Revoke ${revokeTarget?.name || "this credential"}?` :
                dialog === "user" ? "Grant a person access to Alauda." :
                "Create a scoped credential."}
            </DialogDescription>
          </DialogHeader>
          {secret ? (
            <>
              <DialogBody><div className="flex min-w-0 items-center gap-2">
                <code className="min-w-0 flex-1 break-all rounded border border-[var(--border)] bg-[var(--surface-muted)] p-3 text-xs">{secret}</code>
                <Button aria-label="Copy secret" onClick={() => void navigator.clipboard.writeText(secret)}><Copy size={16} /></Button>
              </div></DialogBody>
              <DialogFooter><Button variant="primary" onClick={closeDialog}>Done</Button></DialogFooter>
            </>
          ) : dialog === "revoke" ? (
            <>
              <DialogBody>{error ? <Alert tone="danger" title="Revoke failed">{error}</Alert> : null}
                The selected {revokeTarget?.kind} will no longer grant access.</DialogBody>
              <DialogFooter><Button disabled={busy} onClick={closeDialog}>Cancel</Button><Button variant="danger" disabled={busy} onClick={() => void confirmRevoke()}>Revoke</Button></DialogFooter>
            </>
          ) : (
            <form onSubmit={(event) => void submit(event)}>
              <DialogBody>
                {error ? <Alert tone="danger" title="Unable to create resource">{error}</Alert> : null}
                <div className="grid gap-4">
                  {dialog === "user" ? (
                    <>
                      <FormField label="Username"><Input required autoFocus value={userForm.username} onChange={(event) => setUserForm({ ...userForm, username: event.target.value })} /></FormField>
                      <FormField label="Display name"><Input required value={userForm.displayName} onChange={(event) => setUserForm({ ...userForm, displayName: event.target.value })} /></FormField>
                      <FormField label="Email"><Input required type="email" value={userForm.email} onChange={(event) => setUserForm({ ...userForm, email: event.target.value })} /></FormField>
                      <FormField label="Password"><PasswordInput required value={userForm.password} onChange={(event) => setUserForm({ ...userForm, password: event.target.value })} /></FormField>
                      <FormField label="Role"><Select value={userForm.role} onChange={(event) => setUserForm({ ...userForm, role: event.target.value })}>
                        {["Administrator", "Operator", "Viewer", "Automation"].map((role) => <option key={role}>{role}</option>)}
                      </Select></FormField>
                    </>
                  ) : (
                    <>
                      {dialog === "token" ? <FormField label="Owner"><Select value={ownerId} onChange={(event) => setOwnerId(event.target.value)}>
                        {users.map((user) => <option key={user.id} value={user.id}>{user.username}</option>)}
                      </Select></FormField> : null}
                      <FormField label="Name"><Input required autoFocus value={dialog === "token" ? tokenForm.name : keyForm.name}
                        onChange={(event) => dialog === "token" ? setTokenForm({ ...tokenForm, name: event.target.value }) : setKeyForm({ ...keyForm, name: event.target.value })} /></FormField>
                      <ScopeChoices value={dialog === "token" ? tokenForm.scopes : keyForm.scopes}
                        onChange={(scopes) => dialog === "token" ? setTokenForm({ ...tokenForm, scopes }) : setKeyForm({ ...keyForm, scopes })} />
                      {dialog === "key" ? <FormSection title="Environment access" description="Leave All environments selected for unrestricted access.">
                        <div className="grid gap-2 sm:grid-cols-2">
                          <label className="flex items-center gap-2 text-sm"><Checkbox aria-label="All environments"
                            checked={keyForm.environmentIds.length === 0}
                            onCheckedChange={(checked) => {
                              if (checked) setKeyForm({ ...keyForm, environmentIds: [] });
                              else if (environments[0]) setKeyForm({ ...keyForm, environmentIds: [environments[0].id] });
                            }} />All environments</label>
                          {environments.map((env) => <label className="flex items-center gap-2 text-sm" key={env.id}>
                            <Checkbox aria-label={env.name} checked={keyForm.environmentIds.includes(env.id)}
                              disabled={keyForm.environmentIds.length === 1 && keyForm.environmentIds.includes(env.id)}
                              onCheckedChange={(checked) => setKeyForm({ ...keyForm, environmentIds: checked
                                ? [...keyForm.environmentIds, env.id]
                                : keyForm.environmentIds.filter((id) => id !== env.id) })} />
                            {env.name}</label>)}
                        </div>
                      </FormSection> : null}
                      <FormField label="Expiration"><Select value={dialog === "token" ? tokenForm.expiresInHours : keyForm.expiresInHours}
                        onChange={(event) => dialog === "token" ? setTokenForm({ ...tokenForm, expiresInHours: event.target.value }) : setKeyForm({ ...keyForm, expiresInHours: event.target.value })}>
                        <option value="168">7 days</option><option value="720">30 days</option>
                        <option value="2160">90 days</option><option value="">Never</option>
                      </Select></FormField>
                    </>
                  )}
                </div>
              </DialogBody>
              <DialogFooter>
                <Button type="button" disabled={busy} onClick={closeDialog}>Cancel</Button>
                <Button variant="primary" type="submit" disabled={busy}>{dialog === "user" ? "Create user" : dialog === "token" ? "Create API token" : "Create application key"}</Button>
              </DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>
    </section>
  );
}
