import axios from 'axios';

const api = axios.create({
  baseURL: 'http://localhost:8080/api/v1',
});

export interface Prefix {
  id: string;
  prefix: string;
  description: string;
  scan_enabled?: boolean;
  scan_interval_seconds?: number;
}

export interface IPAddress {
  id: string;
  prefix_id: string;
  ip_address: string;
  status: string;
  description: string;
  last_seen_at?: string | null;
}

export interface Device {
  id: string;
  name: string;
  status: string;
  description: string;
}

export const getPrefixes = () => api.get<Prefix[]>('/prefixes');
export const getIPAddresses = () => api.get<IPAddress[]>('/ip-addresses');
export const getDevices = () => api.get<Device[]>('/devices');

export interface Monitor {
  id: string;
  name: string;
  slug: string;
  monitor_type: 'http' | 'tcp' | 'ping';
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
