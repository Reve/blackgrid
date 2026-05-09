import { useEffect, useState } from 'react';
import { listAuditLog, type AuditLogEntry } from '../api/client';

export default function AuditLogTab() {
  const [entries, setEntries] = useState<AuditLogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filters, setFilters] = useState({ action: '', object_type: '' });

  const load = async () => {
    setLoading(true);
    try {
      const params: Record<string, any> = { limit: 100 };
      if (filters.action) params.action = filters.action;
      if (filters.object_type) params.object_type = filters.object_type;
      setEntries((await listAuditLog(params)).data ?? []);
    } catch (e: any) { setError(e?.message); }
    finally { setLoading(false); }
  };

  useEffect(() => { load(); }, [filters]);

  const actorLabel = (e: AuditLogEntry) => {
    if (e.actor_type === 'system') return <span className="text-text-muted text-xs">system</span>;
    if (e.actor_type === 'api_token') return <span className="text-signal-amber text-xs">token</span>;
    return e.actor_user_id
      ? <span className="text-signal-green text-xs font-mono">{e.actor_user_id.slice(0, 8)}…</span>
      : <span className="text-text-muted text-xs">—</span>;
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg text-signal-green">Audit Log</h3>
        <button className="px-2 py-1 border border-border-color text-text-muted text-xs" onClick={load}>↺ REFRESH</button>
      </div>

      {error && <div className="text-signal-red text-sm">{error}</div>}

      <div className="flex gap-2 text-sm">
        <input
          className="bg-bg-deep border border-border-color p-1 text-text-main text-xs w-40"
          placeholder="Filter action…"
          value={filters.action}
          onChange={e => setFilters({ ...filters, action: e.target.value })}
        />
        <input
          className="bg-bg-deep border border-border-color p-1 text-text-main text-xs w-40"
          placeholder="Filter object type…"
          value={filters.object_type}
          onChange={e => setFilters({ ...filters, object_type: e.target.value })}
        />
      </div>

      {loading ? (
        <div className="text-text-muted text-sm">Loading…</div>
      ) : (
        <table className="w-full text-left border-collapse text-xs">
          <thead><tr className="border-b border-border-color text-text-muted">
            <th className="p-2">Time</th>
            <th className="p-2">Action</th>
            <th className="p-2">Object</th>
            <th className="p-2">Actor</th>
          </tr></thead>
          <tbody>
            {entries.map(e => (
              <tr key={e.id} className="border-b border-border-color hover:bg-surface/30">
                <td className="p-2 text-text-muted font-mono whitespace-nowrap">{new Date(e.created_at).toLocaleString()}</td>
                <td className="p-2 font-mono text-text-main">{e.action}</td>
                <td className="p-2 text-text-muted">
                  {e.object_type && <span>{e.object_type}{e.object_id ? <span className="ml-1 text-text-muted">/{e.object_id.slice(0, 8)}…</span> : ''}</span>}
                </td>
                <td className="p-2">{actorLabel(e)}</td>
              </tr>
            ))}
            {entries.length === 0 && (
              <tr><td colSpan={4} className="p-4 text-text-muted text-center">No audit log entries</td></tr>
            )}
          </tbody>
        </table>
      )}
    </div>
  );
}
