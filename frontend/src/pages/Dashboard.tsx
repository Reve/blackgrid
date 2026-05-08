import { useEffect, useState } from 'react';
import type { Monitor } from "../api/client";
import { getMonitors } from '../api/client';

export default function Dashboard() {
  const [monitors, setMonitors] = useState<Monitor[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const res = await getMonitors();
        setMonitors(res.data);
      } catch (err) {
        console.error(err);
      } finally {
        setLoading(false);
      }
    };
    fetchStats();
  }, []);

  if (loading) return <div className="p-4 text-text-muted">Loading dashboard...</div>;

  const upCount = monitors.filter(m => m.status === 'up').length;
  const downCount = monitors.filter(m => m.status === 'down').length;
  const pausedCount = monitors.filter(m => m.status === 'paused').length;
  const unknownCount = monitors.filter(m => m.status === 'unknown' || m.status === 'degraded').length;

  const recentFailures = monitors.filter(m => m.status === 'down' || m.status === 'degraded').slice(0, 5);

  return (
    <div className="flex flex-col gap-4 h-full">
      <h2 className="text-xl text-signal-green">System Overview</h2>

      <div className="grid grid-cols-4 gap-4">
        <div className="panel flex flex-col items-center justify-center py-6">
          <span className="text-3xl text-signal-green mb-2">{upCount}</span>
          <span className="text-sm text-text-muted uppercase">Up</span>
        </div>
        <div className="panel flex flex-col items-center justify-center py-6 border-signal-red">
          <span className="text-3xl text-signal-red mb-2">{downCount}</span>
          <span className="text-sm text-text-muted uppercase">Down</span>
        </div>
        <div className="panel flex flex-col items-center justify-center py-6">
          <span className="text-3xl text-text-muted mb-2">{pausedCount}</span>
          <span className="text-sm text-text-muted uppercase">Paused</span>
        </div>
        <div className="panel flex flex-col items-center justify-center py-6">
          <span className="text-3xl text-signal-amber mb-2">{unknownCount}</span>
          <span className="text-sm text-text-muted uppercase">Other</span>
        </div>
      </div>

      <div className="flex-1 panel flex flex-col">
        <h3 className="text-lg text-signal-red mb-4">Recent Monitor Failures</h3>
        <div className="flex-1 overflow-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b border-border-color text-text-muted">
                <th className="p-2">Status</th>
                <th className="p-2">Name</th>
                <th className="p-2">Target</th>
                <th className="p-2">Last Checked</th>
              </tr>
            </thead>
            <tbody>
              {recentFailures.map(m => (
                <tr key={m.id} className="border-b border-border-color">
                  <td className="p-2">
                    <span className="text-signal-red uppercase text-xs">{m.status}</span>
                  </td>
                  <td className="p-2 text-text-main">{m.name}</td>
                  <td className="p-2 text-text-muted">{m.target}</td>
                  <td className="p-2 text-text-muted">{m.last_checked_at ? new Date(m.last_checked_at).toLocaleString() : 'Never'}</td>
                </tr>
              ))}
              {recentFailures.length === 0 && (
                <tr>
                  <td colSpan={4} className="p-4 text-center text-text-muted">No recent failures. System is stable.</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
