import axios from 'axios';

const api = axios.create({
  baseURL: 'http://localhost:8080/api/v1',
});

export interface Prefix {
  id: string;
  prefix: string;
  description: string;
}

export interface IPAddress {
  id: string;
  prefix_id: string;
  ip_address: string;
  status: string;
  description: string;
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
