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
