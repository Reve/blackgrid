import axios from 'axios';

const api = axios.create({
  baseURL: '/api/v1',
  withCredentials: true,
});

export interface ApiErrorDetail {
  code: string;
  message: string;
  request_id?: string;
  details?: Record<string, any>;
}

export interface ApiErrorResponse {
  error: ApiErrorDetail;
}

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response && error.response.data && error.response.data.error) {
      // It's our consistent error format
      return Promise.reject(error.response.data.error as ApiErrorDetail);
    }
    // Fallback for other errors (network, etc.)
    return Promise.reject({
      code: 'network_error',
      message: error.message || 'An unexpected error occurred',
    } as ApiErrorDetail);
  }
);

// ---- Existing types ----

export interface Site {
  id: string;
  name: string;
  description: string | null;
  created_at?: string | null;
  updated_at?: string | null;
}

export interface CreateSiteInput {
  name: string;
  description?: string | null;
}

export interface Vlan {
  id: string;
  site_id: string | null;
  vlan_id: number;
  name: string;
  description: string | null;
}

export interface CreateVlanInput {
  site_id?: string | null;
  vlan_id: number;
  name: string;
  description?: string | null;
}

export interface Prefix {
  id: string;
  site_id?: string | null;
  vlan_id?: string | null;
  prefix: string;
  description: string;
  scan_enabled?: boolean;
  scan_interval_seconds?: number;
}

export interface CreatePrefixInput {
  site_id: string;
  vlan_id?: string | null;
  prefix: string;
  description?: string | null;
}

export interface IPAddress {
  id: string;
  prefix_id: string;
  ip_address: string;
  interface_id?: string | null;
  status: string;
  description: string;
  last_seen_at?: string | null;
}

export interface CreateIPAddressInput {
  prefix_id: string;
  ip_address: string;
  interface_id?: string | null;
  status?: string | null;
  description?: string | null;
}

export interface Device {
  id: string;
  name: string;
  site_id?: string | null;
  status: string;
  description: string;
}

export interface CreateDeviceInput {
  name: string;
  site_id?: string | null;
  description?: string | null;
  status?: string | null;
}

// Sites
export const getSites = () => api.get<Site[]>('/sites');
export const createSite = (data: CreateSiteInput) => api.post<Site>('/sites', data);
export const updateSite = (id: string, data: CreateSiteInput) =>
  api.put<Site>(`/sites/${id}`, data);
export const deleteSite = (id: string) => api.delete(`/sites/${id}`);

// VLANs
export const getVlans = () => api.get<Vlan[]>('/vlans');
export const createVlan = (data: CreateVlanInput) => api.post<Vlan>('/vlans', data);
export const updateVlan = (id: string, data: CreateVlanInput) =>
  api.put<Vlan>(`/vlans/${id}`, data);
export const deleteVlan = (id: string) => api.delete(`/vlans/${id}`);

// Prefixes
export const getPrefixes = () => api.get<Prefix[]>('/prefixes');
export const createPrefix = (data: CreatePrefixInput) => api.post<Prefix>('/prefixes', data);
export const updatePrefix = (id: string, data: CreatePrefixInput) =>
  api.put<Prefix>(`/prefixes/${id}`, data);
export const deletePrefix = (id: string) => api.delete(`/prefixes/${id}`);

// IP Addresses
export const getIPAddresses = () => api.get<IPAddress[]>('/ip-addresses');
export const createIPAddress = (data: CreateIPAddressInput) =>
  api.post<IPAddress>('/ip-addresses', data);
export const updateIPAddress = (id: string, data: CreateIPAddressInput) =>
  api.put<IPAddress>(`/ip-addresses/${id}`, data);
export const deleteIPAddress = (id: string) => api.delete(`/ip-addresses/${id}`);
export const reserveIPAddress = (id: string) =>
  api.post<IPAddress>(`/ip-addresses/${id}/reserve`);
export const assignIPAddress = (id: string) =>
  api.post<IPAddress>(`/ip-addresses/${id}/assign`);
export const releaseIPAddress = (id: string) =>
  api.post<IPAddress>(`/ip-addresses/${id}/release`);

// Devices
export const getDevices = () => api.get<Device[]>('/devices');
export const createDevice = (data: CreateDeviceInput) => api.post<Device>('/devices', data);
export const updateDevice = (id: string, data: CreateDeviceInput) =>
  api.put<Device>(`/devices/${id}`, data);
export const deleteDevice = (id: string) => api.delete(`/devices/${id}`);

export interface Monitor {
  id: string;
  name: string;
  slug: string;
  monitor_type: 'http' | 'tcp' | 'ping' | 'dns' | 'tls' | 'push' | 'postgres';
  target: string;
  config: Record<string, any> | null;
  ip_address_id: string | null;
  device_id: string | null;
  interval_seconds: number;
  timeout_seconds: number;
  retry_count: number;
  enabled: boolean;
  status: 'unknown' | 'up' | 'degraded' | 'down' | 'paused';
  last_checked_at: string | null;
  last_status_change_at: string | null;
}

export interface MonitorResult {
  id: string;
  monitor_id: string;
  status: string;
  latency_ms: number | null;
  error_message: string | null;
  details: Record<string, any> | null;
  checked_at: string;
}

export const getMonitors = () => api.get<Monitor[]>('/monitors');
export const getMonitor = (id: string) => api.get<Monitor>(`/monitors/${id}`);
export const createMonitor = (data: Partial<Monitor>) => api.post<Monitor>('/monitors', data);
export const updateMonitor = (id: string, data: Partial<Monitor>) => api.patch<Monitor>(`/monitors/${id}`, data);
export const deleteMonitor = (id: string) => api.delete(`/monitors/${id}`);
export const pauseMonitor = (id: string) => api.post<Monitor>(`/monitors/${id}/pause`);
export const resumeMonitor = (id: string) => api.post<Monitor>(`/monitors/${id}/resume`);
export const testMonitor = (id: string) => api.post<{ status: string, latency_ms: number, error_message?: string }>(`/monitors/${id}/test`);
export const getMonitorResults = (id: string) => api.get<MonitorResult[]>(`/monitors/${id}/results`);
export const rotatePushToken = (id: string) =>
  api.post<{ token: string; message: string; push_url: string }>(`/monitors/${id}/rotate-push-token`);

// Discovery
export type DiscoveryClassification = 'known' | 'new' | 'changed' | 'duplicate' | 'stale' | 'ignored';

export interface DiscoveryScan {
  id: string;
  prefix_id: string;
  status: 'queued' | 'running' | 'completed' | 'failed' | 'cancelled';
  started_at: string | null;
  completed_at: string | null;
  error: string | null;
  created_at: string;
  updated_at: string;
}

export interface DiscoveryResult {
  id: string;
  scan_id: string;
  prefix_id: string;
  address: string;
  mac_address: string | null;
  hostname: string | null;
  reverse_dns: string | null;
  open_ports: number[] | null;
  latency_ms: number | null;
  classification: DiscoveryClassification;
  seen_at: string;
  ignored: boolean;
  accepted_at: string | null;
  created_ip_address_id: string | null;
}

export interface DiscoveryResultsFilters {
  scan_id?: string;
  prefix_id?: string;
  classification?: DiscoveryClassification;
  ignored?: boolean;
  limit?: number;
  offset?: number;
}

export const listDiscoveryScans = (params?: { prefix_id?: string; status?: string; limit?: number; offset?: number }) =>
  api.get<DiscoveryScan[]>('/discovery/scans', { params });

export const getDiscoveryScan = (id: string) => api.get<DiscoveryScan>(`/discovery/scans/${id}`);

export const startDiscoveryScan = (prefix_id: string) =>
  api.post<DiscoveryScan>('/discovery/scans', { prefix_id });

export const startPrefixScan = (prefix_id: string) =>
  api.post<DiscoveryScan>(`/prefixes/${prefix_id}/scan`);

export const listDiscoveryResults = (params?: DiscoveryResultsFilters) =>
  api.get<DiscoveryResult[]>('/discovery/results', { params });

export const acceptDiscoveryResult = (id: string, body: { hostname?: string; fqdn?: string; status?: string }) =>
  api.post<IPAddress>(`/discovery/results/${id}/accept`, body);

export const ignoreDiscoveryResult = (id: string) =>
  api.post<DiscoveryResult>(`/discovery/results/${id}/ignore`);

export const updatePrefixScanConfig = (id: string, body: { scan_enabled: boolean; scan_interval_seconds: number }) =>
  api.put<Prefix>(`/prefixes/${id}/scan-config`, body);

export interface DiscoveryDiagnostics {
  worker_count: number;
  default_ports: number[];
  tcp_timeout_ms: number;
  ping_supported: boolean;
  runtime: {
    inside_container: boolean;
    hostname: string;
    routes?: string[];
  };
}

export interface DiscoveryProbeRequest {
  address: string;
  ports?: number[];
}

export interface DiscoveryProbeResponse {
  address: string;
  seen: boolean;
  open_ports: number[];
  latency_ms: number;
  reverse_dns: string;
}

export const getDiscoveryDiagnostics = () =>
  api.get<DiscoveryDiagnostics>('/discovery/diagnostics');

export const probeDiscoveryHost = (req: DiscoveryProbeRequest) =>
  api.post<DiscoveryProbeResponse>('/discovery/probe', req);

// Incidents
export type IncidentStatus = 'open' | 'acknowledged' | 'resolved';
export type IncidentSeverity = 'info' | 'warning' | 'critical';

export interface Incident {
  id: string;
  monitor_id: string;
  status: IncidentStatus;
  severity: IncidentSeverity;
  started_at: string | null;
  acknowledged_at: string | null;
  resolved_at: string | null;
  summary: string;
  details: string | null;
  created_at: string | null;
  updated_at: string | null;
}

export interface IncidentCounts {
  open_count: number;
  acknowledged_count: number;
  critical_count: number;
  resolved_24h_count: number;
}

export interface ListIncidentsParams {
  status?: IncidentStatus;
  severity?: IncidentSeverity;
  monitor_id?: string;
  limit?: number;
  offset?: number;
}

export const listIncidents = (params?: ListIncidentsParams) =>
  api.get<Incident[]>('/incidents', { params });
export const getIncident = (id: string) => api.get<Incident>(`/incidents/${id}`);
export const getIncidentCounts = () => api.get<IncidentCounts>('/incidents/counts');
export const acknowledgeIncident = (id: string, note?: string) =>
  api.post<Incident>(`/incidents/${id}/acknowledge`, { note: note ?? '' });
export const resolveIncident = (id: string, note?: string) =>
  api.post<Incident>(`/incidents/${id}/resolve`, { note: note ?? '' });

// Notification channels
export type ChannelType = 'webhook' | 'smtp';

export interface NotificationChannel {
  id: string;
  name: string;
  channel_type: ChannelType;
  enabled: boolean;
  config: Record<string, any>;
  created_at: string | null;
  updated_at: string | null;
}

export interface CreateChannelInput {
  name: string;
  channel_type: ChannelType;
  enabled?: boolean;
  config: Record<string, any>;
}

export const listNotificationChannels = () =>
  api.get<NotificationChannel[]>('/notification-channels');
export const createNotificationChannel = (data: CreateChannelInput) =>
  api.post<NotificationChannel>('/notification-channels', data);
export const updateNotificationChannel = (id: string, data: Partial<CreateChannelInput>) =>
  api.patch<NotificationChannel>(`/notification-channels/${id}`, data);
export const deleteNotificationChannel = (id: string) =>
  api.delete(`/notification-channels/${id}`);
export const testNotificationChannel = (id: string) =>
  api.post<{ status: string; event_type: string; error?: string; sent_at?: string }>(
    `/notification-channels/${id}/test`,
  );

// Status pages
export interface StatusPage {
  id: string;
  name: string;
  slug: string;
  description: string | null;
  public: boolean;
  show_uptime: boolean;
  show_incidents: boolean;
  created_at: string | null;
  updated_at: string | null;
}

export interface AttachedMonitor {
  monitor: Monitor;
  display_name: string | null;
  display_order: number;
  created_at: string | null;
}

export interface AdminStatusPage {
  page: StatusPage;
  monitors: AttachedMonitor[];
}

export interface CreateStatusPageInput {
  name: string;
  slug?: string;
  description?: string | null;
  public?: boolean;
  show_uptime?: boolean;
  show_incidents?: boolean;
}

export interface AttachStatusPageMonitorInput {
  monitor_id: string;
  display_name?: string;
  display_order?: number;
}

export type AggregateStatus = 'up' | 'degraded' | 'down' | 'empty';

export interface PublicStatusMonitor {
  display_name: string;
  monitor_type: string;
  status: string;
  last_checked_at: string | null;
  uptime_24h: number | null;
  uptime_30d: number | null;
}

export interface PublicIncident {
  monitor_display_name: string;
  severity: string;
  status: string;
  started_at: string | null;
  resolved_at: string | null;
  summary: string;
}

export interface PublicStatusPageResponse {
  name: string;
  slug: string;
  description: string;
  aggregate_status: AggregateStatus;
  monitors: PublicStatusMonitor[];
  incidents?: PublicIncident[];
}

export const listStatusPages = () => api.get<StatusPage[]>('/status-pages');
export const getStatusPage = (id: string) => api.get<AdminStatusPage>(`/status-pages/${id}`);
export const createStatusPage = (data: CreateStatusPageInput) =>
  api.post<StatusPage>('/status-pages', data);
export const updateStatusPage = (id: string, data: Partial<CreateStatusPageInput>) =>
  api.patch<StatusPage>(`/status-pages/${id}`, data);
export const deleteStatusPage = (id: string) => api.delete(`/status-pages/${id}`);
export const attachStatusPageMonitor = (id: string, data: AttachStatusPageMonitorInput) =>
  api.post<AttachedMonitor>(`/status-pages/${id}/monitors`, data);
export const updateAttachedStatusPageMonitor = (
  id: string,
  monitorId: string,
  data: { display_name?: string; display_order?: number },
) => api.patch<AttachedMonitor>(`/status-pages/${id}/monitors/${monitorId}`, data);
export const removeAttachedStatusPageMonitor = (id: string, monitorId: string) =>
  api.delete(`/status-pages/${id}/monitors/${monitorId}`);
export const reorderStatusPageMonitors = (id: string, monitorIds: string[]) =>
  api.post(`/status-pages/${id}/monitors/reorder`, { monitor_ids: monitorIds });

// Public status page is at the root, not under /api/v1.
const publicAxios = axios.create({ baseURL: '' });

publicAxios.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response && error.response.data && error.response.data.error) {
      return Promise.reject(error.response.data.error as ApiErrorDetail);
    }
    return Promise.reject({
      code: 'network_error',
      message: error.message || 'An unexpected error occurred',
    } as ApiErrorDetail);
  }
);
export const getPublicStatusPage = (slug: string) =>
  publicAxios.get<PublicStatusPageResponse>(`/status/${slug}`);

// ---- Phase 6: Auth ----

export type UserRole = 'admin' | 'operator' | 'viewer';

export interface AuthUser {
  id: string;
  email: string;
  display_name: string;
  role: UserRole;
  enabled: boolean;
  last_login_at: string | null;
  created_at: string | null;
  updated_at: string | null;
}

export interface SetupStatus {
  setup_required: boolean;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface SetupAdminRequest {
  email: string;
  display_name: string;
  password: string;
}

export interface CreateUserRequest {
  email: string;
  display_name: string;
  password: string;
  role: UserRole;
  enabled?: boolean;
}

export interface UpdateUserRequest {
  display_name?: string;
  role?: UserRole;
  enabled?: boolean;
  password?: string;
}

export interface ApiToken {
  id: string;
  user_id: string;
  name: string;
  role: UserRole;
  last_used_at: string | null;
  expires_at: string | null;
  created_at: string;
}

export interface CreateApiTokenRequest {
  user_id: string;
  name: string;
  role: UserRole;
  expires_at?: string | null;
}

export interface CreateApiTokenResponse {
  token: string; // plaintext — shown once
  api_token: ApiToken;
}

export interface AuditLogEntry {
  id: string;
  action: string;
  entity_type: string;
  entity_id: string;
  actor_user_id: string | null;
  actor_type: string | null;
  actor_api_token_id: string | null;
  object_type: string | null;
  object_id: string | null;
  before_state: any;
  after_state: any;
  created_at: string;
}

export const getSetupStatus = () => api.get<SetupStatus>('/setup/status');
export const setupAdmin = (data: SetupAdminRequest) => api.post<AuthUser>('/setup/admin', data);
export const login = (data: LoginRequest) =>
  api.post<{ user: AuthUser }>('/auth/login', data);
export const logout = () => api.post('/auth/logout');
export const getMe = () => api.get<{ user: AuthUser }>('/auth/me');

export const listUsers = () => api.get<AuthUser[]>('/users');
export const createUser = (data: CreateUserRequest) => api.post<AuthUser>('/users', data);
export const getUser = (id: string) => api.get<AuthUser>(`/users/${id}`);
export const updateUser = (id: string, data: UpdateUserRequest) =>
  api.patch<AuthUser>(`/users/${id}`, data);
export const deleteUser = (id: string) => api.delete(`/users/${id}`);

export const listApiTokens = () => api.get<ApiToken[]>('/api-tokens');
export const createApiToken = (data: CreateApiTokenRequest) =>
  api.post<CreateApiTokenResponse>('/api-tokens', data);
export const deleteApiToken = (id: string) => api.delete(`/api-tokens/${id}`);

export interface HealthInfo {
  status: string;
  version: string;
  commit: string;
  build_date: string;
}

export const getHealth = () => api.get<HealthInfo>('/health');

export interface Diagnostics {
  version: { version: string; commit: string; build_date: string };
  database: { status: string; error: string };
  monitor_scheduler: {
    running: boolean;
    worker_count: number;
    last_tick_at: string | null;
    next_due_at: string | null;
  };
  discovery_scheduler: {
    running: boolean;
    worker_count: number;
    last_tick_at: string | null;
    running_scans: number;
  };
  events: { sse_clients: number };
  retention: {
    monitor_results_days: number;
    notification_deliveries_days: number;
    audit_log_days: number;
    discovery_results_days: number;
    discovery_scans_days: number;
    interval_hours: number;
  };
  current_user_role: string;
  server_time: string;
}

export const getDiagnostics = () => api.get<Diagnostics>('/admin/diagnostics');

export const listAuditLog = (params?: {
  actor_user_id?: string;
  action?: string;
  object_type?: string;
  object_id?: string;
  limit?: number;
  offset?: number;
}) => api.get<AuditLogEntry[]>('/audit-log', { params });
