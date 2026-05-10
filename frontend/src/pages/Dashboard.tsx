import { useEffect, useState } from 'react';
import type { Incident, IncidentCounts, Monitor, StatusPage } from '../api/client';
import { getIncidentCounts, getMonitors, listIncidents, listStatusPages } from '../api/client';
import { useEvents } from '../context/EventContext';
import type { Event } from '../lib/events/types';
import { Loading, ErrorState } from '../components/UI';
import OnboardingChecklist from '../components/OnboardingChecklist';
import type { ApiErrorDetail } from '../api/client';

export default function Dashboard() {
  const [monitors, setMonitors] = useState<Monitor[]>([]);
  const [counts, setCounts] = useState<IncidentCounts | null>(null);
  const [activeIncidents, setActiveIncidents] = useState<Incident[]>([]);
  const [recentResolved, setRecentResolved] = useState<Incident[]>([]);
  const [statusPages, setStatusPages] = useState<StatusPage[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<ApiErrorDetail | null>(null);
  const { subscribe, lastEvents } = useEvents();

  const load = async () => {
    try {
      const [m, c, active, resolved, sp] = await Promise.all([
        getMonitors(),
        getIncidentCounts(),
        listIncidents({ status: 'open', limit: 10 }),
        listIncidents({ status: 'resolved', limit: 10 }),
        listStatusPages(),
      ]);
      setMonitors(m.data ?? []);
      setCounts(c.data);
      setActiveIncidents(active.data ?? []);
      setRecentResolved(resolved.data ?? []);
      setStatusPages(sp.data ?? []);
      setError(null);
    } catch (err: any) {
      console.error(err);
      setError(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    const t = setInterval(load, 60000); // Polling reduced as we have events

    const unsubscribe = subscribe((event: Event) => {
      // Reload relevant data on change events
      if (
        event.type.includes('monitor') || 
        event.type.includes('incident') || 
        event.type.includes('status_page') ||
        event.type.includes('discovery')
      ) {
        load();
      }
    });

    return () => {
      clearInterval(t);
      unsubscribe();
    };
  }, [subscribe]);

  if (loading) return <Loading message="Syncing Dashboard..." />;
  if (error) return <ErrorState error={error} onRetry={load} />;

  const monitorById: Record<string, Monitor> = {};
  for (const m of monitors) monitorById[m.id] = m;

  const upCount = monitors.filter(m => m.status === 'up').length;
  const downCount = monitors.filter(m => m.status === 'down').length;
  const pausedCount = monitors.filter(m => m.status === 'paused').length;

  // Breakdown by type
  const typeCounts = monitors.reduce((acc, m) => {
      acc[m.monitor_type] = (acc[m.monitor_type] || 0) + 1;
      return acc;
  }, {} as Record<string, number>);

  return (
    <div className="flex flex-col gap-4 h-full overflow-auto">
      <h2 className="text-xl text-signal-green">System Overview</h2>

      <OnboardingChecklist />

      <div className="grid grid-cols-4 gap-4">
        <Card label="Active Incidents" value={counts?.open_count ?? 0} accent="text-signal-red" />
        <Card label="Critical" value={counts?.critical_count ?? 0} accent="text-signal-red" border="border-signal-red" />
        <Card label="Acknowledged" value={counts?.acknowledged_count ?? 0} accent="text-signal-amber" />
        <Card label="Resolved 24h" value={counts?.resolved_24h_count ?? 0} accent="text-signal-green" />
      </div>

      <div className="grid grid-cols-5 gap-4">
        <Card label="Monitors Up" value={upCount} accent="text-signal-green" />
        <Card label="Monitors Down" value={downCount} accent="text-signal-red" />
        <Card label="Paused" value={pausedCount} accent="text-text-muted" />
        <Card label="Total Monitors" value={monitors.length} accent="text-text-main" />
        <Card label="Status Pages" value={statusPages.length} accent="text-signal-green" />
      </div>

      <div className="grid grid-cols-3 gap-4">
          <div className="panel col-span-1">
              <h3 className="text-sm text-text-muted uppercase mb-3">Monitor Types</h3>
              <div className="flex flex-wrap gap-x-4 gap-y-2">
                  {Object.entries(typeCounts).map(([type, count]) => (
                      <div key={type} className="flex gap-2 text-xs">
                          <span className="text-text-muted uppercase">{type}:</span>
                          <span className="text-text-main font-bold">{count}</span>
                      </div>
                  ))}
              </div>
          </div>
          <div className="panel col-span-2">
              <h3 className="text-sm text-signal-amber uppercase mb-3">TLS Certificates Expiring Soon</h3>
              <div className="text-xs text-text-muted italic">
                  Feature coming soon (requires detailed result scanning).
              </div>
          </div>
      </div>

      <div className="panel">
        <h3 className="text-lg text-signal-green mb-3">Public Status Pages</h3>
        {statusPages.filter(p => p.public).length === 0 ? (
          <p className="text-text-muted text-sm">No public status pages configured.</p>
        ) : (
          <table className="w-full text-left border-collapse text-sm">
            <thead>
              <tr className="border-b border-border-color text-text-muted">
                <th className="p-2">Name</th>
                <th className="p-2">Slug</th>
                <th className="p-2">Public link</th>
              </tr>
            </thead>
            <tbody>
              {statusPages.filter(p => p.public).map(p => (
                <tr key={p.id} className="border-b border-border-color">
                  <td className="p-2 text-text-main">{p.name}</td>
                  <td className="p-2 text-text-muted font-mono text-xs">{p.slug}</td>
                  <td className="p-2 text-xs">
                    <a href={`/status/${p.slug}`} target="_blank" rel="noreferrer"
                       className="text-signal-green hover:underline">/status/{p.slug}</a>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <div className="grid grid-cols-3 gap-4 flex-1 min-h-0">
        <div className="panel flex flex-col col-span-1">
          <h3 className="text-lg text-signal-red mb-4">Active Incident Feed</h3>
          <IncidentTable
            incidents={activeIncidents}
            monitorById={monitorById}
            emptyMessage="No active incidents."
          />
        </div>
        <div className="panel flex flex-col col-span-1">
          <h3 className="text-lg text-text-muted mb-4">Recent Resolved</h3>
          <IncidentTable
            incidents={recentResolved}
            monitorById={monitorById}
            emptyMessage="No resolved incidents."
            muted
          />
        </div>
        <div className="panel flex flex-col col-span-1">
          <h3 className="text-lg text-signal-green mb-4">Live Event Feed</h3>
          <EventFeed events={lastEvents} />
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
function EventFeed({ events }: { events: Event[] }) {
  if (events.length === 0) {
    return <div className="flex-1 flex items-center justify-center text-text-muted text-sm italic">Waiting for events...</div>;
  }
  return (
    <div className="flex-1 overflow-auto">
      <div className="space-y-2">
        {events.map((e) => (
          <div key={e.id} className="text-xs border-l-2 border-surface pl-2 py-1">
            <div className="flex justify-between text-text-muted mb-1">
              <span className="uppercase font-bold text-[10px] tracking-tighter">
                {e.type.replace('.', '_')}
              </span>
              <span>{new Date(e.timestamp).toLocaleTimeString()}</span>
            </div>
            <div className="text-text-main truncate">
              {e.payload.name || e.payload.summary || e.payload.address || e.payload.action || 'Event trigger'}
            </div>
            {e.payload.new_status && (
              <div className="text-[10px]">
                STATUS: <span className={e.payload.new_status === 'up' ? 'text-signal-green' : 'text-signal-red'}>{e.payload.new_status.toUpperCase()}</span>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
