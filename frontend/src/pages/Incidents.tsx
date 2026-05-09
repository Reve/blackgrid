import { useEffect, useMemo, useState } from 'react';
import {
  acknowledgeIncident,
  getMonitors,
  listIncidents,
  resolveIncident,
} from '../api/client';
import type {
  Incident,
  IncidentSeverity,
  IncidentStatus,
  Monitor,
} from '../api/client';

const STATUS_FILTERS: ('all' | IncidentStatus)[] = ['all', 'open', 'acknowledged', 'resolved'];
const SEVERITY_FILTERS: ('all' | IncidentSeverity)[] = ['all', 'critical', 'warning', 'info'];

function severityClasses(sev: IncidentSeverity, resolved: boolean): string {
  if (resolved) return 'text-text-muted border-border-color opacity-70';
  switch (sev) {
    case 'critical':
      return 'text-signal-red border-signal-red';
    case 'warning':
      return 'text-signal-amber border-signal-amber';
    default:
      return 'text-signal-green border-signal-green';
  }
}

function fmt(ts: string | null | undefined): string {
  if (!ts) return '—';
  return new Date(ts).toLocaleString();
}

export default function Incidents() {
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [monitors, setMonitors] = useState<Record<string, Monitor>>({});
  const [statusFilter, setStatusFilter] = useState<'all' | IncidentStatus>('all');
  const [severityFilter, setSeverityFilter] = useState<'all' | IncidentSeverity>('all');
  const [selected, setSelected] = useState<Incident | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = async () => {
    setLoading(true);
    try {
      const [incRes, monRes] = await Promise.all([listIncidents(), getMonitors()]);
      setIncidents(incRes.data ?? []);
      const map: Record<string, Monitor> = {};
      for (const m of monRes.data ?? []) map[m.id] = m;
      setMonitors(map);
      setError(null);
    } catch (err: any) {
      setError(err?.message ?? 'failed to load incidents');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    const t = setInterval(load, 15000);
    return () => clearInterval(t);
  }, []);

  const filtered = useMemo(() => {
    return incidents.filter(i => {
      if (statusFilter !== 'all' && i.status !== statusFilter) return false;
      if (severityFilter !== 'all' && i.severity !== severityFilter) return false;
      return true;
    });
  }, [incidents, statusFilter, severityFilter]);

  const groupedActive = filtered.filter(i => i.status === 'open');
  const groupedAck = filtered.filter(i => i.status === 'acknowledged');
  const groupedResolved = filtered.filter(i => i.status === 'resolved');

  const onAck = async (id: string) => {
    try {
      await acknowledgeIncident(id);
      await load();
    } catch (err: any) {
      alert(err?.response?.data?.error ?? 'acknowledge failed');
    }
  };

  const onResolve = async (id: string) => {
    try {
      await resolveIncident(id);
      await load();
    } catch (err: any) {
      alert(err?.response?.data?.error ?? 'resolve failed');
    }
  };

  return (
    <div className="flex flex-col gap-4 h-full">
      <div className="flex items-center justify-between">
        <h2 className="text-xl text-signal-green">Incidents</h2>
        <div className="flex gap-2 text-xs">
          <select
            className="bg-bg-deep border border-border-color text-text-main p-1"
            value={statusFilter}
            onChange={e => setStatusFilter(e.target.value as any)}
          >
            {STATUS_FILTERS.map(s => <option key={s} value={s}>{s.toUpperCase()}</option>)}
          </select>
          <select
            className="bg-bg-deep border border-border-color text-text-main p-1"
            value={severityFilter}
            onChange={e => setSeverityFilter(e.target.value as any)}
          >
            {SEVERITY_FILTERS.map(s => <option key={s} value={s}>{s.toUpperCase()}</option>)}
          </select>
          <button onClick={load} className="px-2 border border-border-color text-text-muted hover:text-signal-green">
            REFRESH
          </button>
        </div>
      </div>

      {error && <div className="panel text-signal-red">{error}</div>}
      {loading && <div className="text-text-muted">Loading incidents...</div>}

      <div className="grid grid-cols-3 gap-4 flex-1 min-h-0">
        <div className="col-span-2 flex flex-col gap-4 overflow-auto">
          <IncidentSection
            title="ACTIVE"
            color="text-signal-red"
            items={groupedActive}
            monitors={monitors}
            onSelect={setSelected}
            onAck={onAck}
            onResolve={onResolve}
          />
          <IncidentSection
            title="ACKNOWLEDGED"
            color="text-signal-amber"
            items={groupedAck}
            monitors={monitors}
            onSelect={setSelected}
            onAck={onAck}
            onResolve={onResolve}
          />
          <IncidentSection
            title="RESOLVED"
            color="text-text-muted"
            items={groupedResolved}
            monitors={monitors}
            onSelect={setSelected}
            onAck={onAck}
            onResolve={onResolve}
          />
        </div>

        <div className="panel overflow-auto">
          <h3 className="text-lg text-signal-green mb-4">Detail</h3>
          {selected ? (
            <IncidentDetail incident={selected} monitor={monitors[selected.monitor_id]} onAck={onAck} onResolve={onResolve} />
          ) : (
            <p className="text-text-muted text-sm">Select an incident to inspect it.</p>
          )}
        </div>
      </div>
    </div>
  );
}

interface SectionProps {
  title: string;
  color: string;
  items: Incident[];
  monitors: Record<string, Monitor>;
  onSelect: (i: Incident) => void;
  onAck: (id: string) => void;
  onResolve: (id: string) => void;
}

function IncidentSection({ title, color, items, monitors, onSelect, onAck, onResolve }: SectionProps) {
  return (
    <div className="panel">
      <h3 className={`text-lg mb-3 ${color}`}>{title} ({items.length})</h3>
      {items.length === 0 ? (
        <p className="text-text-muted text-sm">None.</p>
      ) : (
        <table className="w-full text-left border-collapse text-sm">
          <thead>
            <tr className="border-b border-border-color text-text-muted">
              <th className="p-2">Sev</th>
              <th className="p-2">Monitor</th>
              <th className="p-2">Summary</th>
              <th className="p-2">Started</th>
              <th className="p-2">Actions</th>
            </tr>
          </thead>
          <tbody>
            {items.map(i => {
              const resolved = i.status === 'resolved';
              const cls = severityClasses(i.severity, resolved);
              const mon = monitors[i.monitor_id];
              return (
                <tr
                  key={i.id}
                  className={`border-b border-border-color cursor-pointer hover:bg-bg-deep ${resolved ? 'opacity-70' : ''}`}
                  onClick={() => onSelect(i)}
                >
                  <td className="p-2">
                    <span className={`px-2 py-0.5 border ${cls} uppercase text-xs`}>{i.severity}</span>
                  </td>
                  <td className="p-2 text-text-main">{mon?.name ?? i.monitor_id.slice(0, 8)}</td>
                  <td className="p-2 text-text-muted">{i.summary}</td>
                  <td className="p-2 text-text-muted">{fmt(i.started_at)}</td>
                  <td className="p-2">
                    <div className="flex gap-1" onClick={e => e.stopPropagation()}>
                      {i.status === 'open' && (
                        <button className="text-xs px-2 border border-signal-amber text-signal-amber" onClick={() => onAck(i.id)}>
                          ACK
                        </button>
                      )}
                      {i.status !== 'resolved' && (
                        <button className="text-xs px-2 border border-signal-green text-signal-green" onClick={() => onResolve(i.id)}>
                          RESOLVE
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      )}
    </div>
  );
}

function IncidentDetail({
  incident,
  monitor,
  onAck,
  onResolve,
}: {
  incident: Incident;
  monitor?: Monitor;
  onAck: (id: string) => void;
  onResolve: (id: string) => void;
}) {
  const cls = severityClasses(incident.severity, incident.status === 'resolved');
  return (
    <div className="text-sm flex flex-col gap-2">
      <div>
        <span className={`px-2 py-0.5 border ${cls} uppercase text-xs mr-2`}>{incident.severity}</span>
        <span className="text-text-muted uppercase text-xs">{incident.status}</span>
      </div>
      <div><span className="text-text-muted">ID:</span> <span className="text-text-main">{incident.id}</span></div>
      <div><span className="text-text-muted">Monitor:</span> <span className="text-text-main">{monitor?.name ?? incident.monitor_id}</span></div>
      <div><span className="text-text-muted">Summary:</span> <span className="text-text-main">{incident.summary}</span></div>
      {incident.details && <div className="text-text-muted whitespace-pre-wrap">{incident.details}</div>}
      <div><span className="text-text-muted">Started:</span> {fmt(incident.started_at)}</div>
      <div><span className="text-text-muted">Acknowledged:</span> {fmt(incident.acknowledged_at)}</div>
      <div><span className="text-text-muted">Resolved:</span> {fmt(incident.resolved_at)}</div>
      <div className="flex gap-2 pt-2">
        {incident.status === 'open' && (
          <button className="px-2 py-1 border border-signal-amber text-signal-amber text-xs" onClick={() => onAck(incident.id)}>
            ACKNOWLEDGE
          </button>
        )}
        {incident.status !== 'resolved' && (
          <button className="px-2 py-1 border border-signal-green text-signal-green text-xs" onClick={() => onResolve(incident.id)}>
            RESOLVE
          </button>
        )}
        {monitor && (
          <a href={`/monitors/${monitor.id}`} className="px-2 py-1 border border-border-color text-text-muted hover:text-signal-green text-xs">
            OPEN MONITOR
          </a>
        )}
      </div>
    </div>
  );
}
