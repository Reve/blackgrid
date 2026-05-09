import { useEffect, useState } from 'react';
import type { Device } from "../api/client";
import { getDevices } from '../api/client';

export default function Devices() {
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchDevices = async () => {
      try {
        const res = await getDevices();
        setDevices(res.data || []);
      } catch (error) {
        console.error("Failed to fetch devices", error);
      } finally {
        setLoading(false);
      }
    };
    fetchDevices();
  }, []);

  if (loading) return <div className="p-4 text-signal-amber">Loading devices...</div>;

  return (
    <div className="panel">
      <h2 className="text-xl text-signal-green mb-4">Devices</h2>
      <div className="overflow-x-auto">
        <table className="w-full text-left border-collapse">
          <thead>
            <tr className="border-b border-surface">
              <th className="p-2">Name</th>
              <th className="p-2">Status</th>
              <th className="p-2">Description</th>
            </tr>
          </thead>
          <tbody>
            {devices.length === 0 ? (
              <tr><td colSpan={3} className="p-2 text-text-muted">No devices found</td></tr>
            ) : devices.map(d => (
              <tr key={d.id} className="border-b border-surface">
                <td className="p-2 text-signal-green">{d.name}</td>
                <td className="p-2">
                  <span className={`px-2 py-1 rounded text-xs ${d.status === 'active' ? 'bg-signal-green/20 text-signal-green' : 'bg-surface text-text-muted'}`}>
                    {d.status}
                  </span>
                </td>
                <td className="p-2 text-text-muted">{d.description}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
