import { useEffect, useState } from 'react';
import type { Incident, IncidentCounts, Monitor } from '../api/client';
import { getIncidentCounts, getMonitors, listIncidents } from '../api/client';

export default function Dashboard() {
  const [monitors, setMonitors] = useState<Monitor[]>([]);
  const [counts, setCounts] = useState<IncidentCounts | null>(null);
  const [activeIncidents, setActiveIncidents] = useState<Incident[]>([]);
  const [recentResolved, setRecentResolved] = useState<Incident[]>([]);
  const [loading, setLoading] = useState(true);

  const load = async () => {
    try {
      const [m, c, active, resolved] = await Promise.all([
        getMonitors(),
        getIncidentCounts(),
        listIncidents({ status: 'open', limit: 10 }),
        listIncidents({ status: 'resolved', limit: 10 }),
      ]);
      setMonitors(m.data ?? []);
      setCounts(c.data);
      setActiveIncidents(active.data ?? []);
      setRecentResolved(resolved.data ?? []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    const t = setInterval(load, 15000);
    return () => clearInterval(t);
  }, []);

  if (loading) return <div className="p-4 text-text-muted">Loading dashboard...</div>;

  const monitorById: Record<string, Monitor> = {};
  for (const m of monitors) monitorById[m.id] = m;

  const upCount = monitors.filter(m => m.status === 'up').length;
  const downCount = monitors.filter(m => m.status === 'down').length;
  const pausedCount = monitors.filter(m => m.status === 'paused').length;

  return (
    <div className="flex flex-col gap-4 h-full">
      <h2 className="text-xl text-signal-green">System Overview</h2>

      <div className="grid grid-cols-4 gap-4">
        <Card label="Active Incidents" value={counts?.open_count ?? 0} accent="text-signal-red" />
        <Card label="Critical" value={counts?.critical_count ?? 0} accent="text-signal-red" border="border-signal-red" />
        <Card label="Acknowledged" value={counts?.acknowledged_count ?? 0} accent="text-signal-amber" />
        <Card label="Resolved 24h" value={counts?.resolved_24h_count ?? 0} accent="text-signal-green" />
      </div>

      <div className="grid grid-cols-4 gap-4">
        <Card label="Monitors Up" value={upCount} accent="text-signal-green" />
        <Card label="Monitors Down" value={downCount} accent="text-signal-red" />
        <Card label="Paused" value={pausedCount} accent="text-text-muted" />
        <Card label="Total Monitors" value={monitors.length} accent="text-text-main" />
      </div>

      <div className="grid grid-cols-2 gap-4 flex-1 min-h-0">
        <div className="panel flex flex-col">
          <h3 className="text-lg text-signal-red mb-4">Active Incident Feed</h3>
          <IncidentTable
            incidents={activeIncidents}
            monitorById={monitorById}
            emptyMessage="No active incidents. System is stable."
          />
        </div>
        <div className="panel flex flex-col">
          <h3 className="text-lg text-text-muted mb-4">Recent Resolved Incidents</h3>
          <IncidentTable
            incidents={recentResolved}
            monitorById={monitorById}
            emptyMessage="No resolved incidents recorded."
            muted
          />
        </div>
      </div>
    </div>
  );
}

function Card({ label, value, accent, border }: { label: string; value: number; accent: string; border?: string }) {
  return (
    <div className={`panel flex flex-col items-center justify-center py-6 ${border ?? ''}`}>
      <span className={`text-3xl mb-2 ${accent}`}>{value}</span>
      <span className="text-sm text-text-muted uppercase">{label}</span>
    </div>
  );
}

function IncidentTable({
  incidents,
  monitorById,
  emptyMessage,
  muted,
}: {
  incidents: Incident[];
  monitorById: Record<string, Monitor>;
  emptyMessage: string;
  muted?: boolean;
}) {
  if (incidents.length === 0) {
    return <div className="flex-1 flex items-center justify-center text-text-muted text-sm">{emptyMessage}</div>;
  }
  return (
    <div className="flex-1 overflow-auto">
      <table className="w-full text-left border-collapse text-sm">
        <thead>
          <tr className="border-b border-border-color text-text-muted">
            <th className="p-2">Sev</th>
            <th className="p-2">Monitor</th>
            <th className="p-2">Summary</th>
            <th className="p-2">When</th>
          </tr>
        </thead>
        <tbody>
          {incidents.map(i => {
            const sevColor =
              i.severity === 'critical' ? 'text-signal-red' :
              i.severity === 'warning' ? 'text-signal-amber' :
              'text-signal-green';
            const monName = monitorById[i.monitor_id]?.name ?? i.monitor_id.slice(0, 8);
            const when = i.status === 'resolved' ? i.resolved_at : i.started_at;
            return (
              <tr key={i.id} className={`border-b border-border-color ${muted ? 'opacity-70' : ''}`}>
                <td className="p-2">
                  <span className={`uppercase text-xs ${sevColor}`}>{i.severity}</span>
                </td>
                <td className="p-2 text-text-main">{monName}</td>
                <td className="p-2 text-text-muted">{i.summary}</td>
                <td className="p-2 text-text-muted">{when ? new Date(when).toLocaleString() : '—'}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
