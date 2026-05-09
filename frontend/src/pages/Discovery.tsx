import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  acceptDiscoveryResult,
  getPrefixes,
  ignoreDiscoveryResult,
  listDiscoveryResults,
  listDiscoveryScans,
  startDiscoveryScan,
  type DiscoveryClassification,
  type DiscoveryResult,
  type DiscoveryScan,
  type Prefix,
} from '../api/client';
import { useEvents } from '../context/EventContext';

const CLASSIFICATIONS: DiscoveryClassification[] = ['known', 'new', 'changed', 'duplicate', 'stale', 'ignored'];

const classificationStyle: Record<DiscoveryClassification, string> = {
  known: 'text-signal-green',
  new: 'text-signal-amber',
  changed: 'text-signal-amber',
  duplicate: 'text-signal-red',
  stale: 'text-text-muted',
  ignored: 'text-text-muted line-through',
};

function classificationTag(c: DiscoveryClassification) {
  return `[${c.toUpperCase()}]`;
}

function formatDuration(scan: DiscoveryScan): string {
  if (!scan.started_at) return '-';
  const start = new Date(scan.started_at).getTime();
  const end = scan.completed_at ? new Date(scan.completed_at).getTime() : Date.now();
  const ms = end - start;
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60_000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.floor(ms / 60_000)}m ${Math.floor((ms % 60_000) / 1000)}s`;
}

export default function Discovery() {
  const navigate = useNavigate();
  const [prefixes, setPrefixes] = useState<Prefix[]>([]);
  const [scans, setScans] = useState<DiscoveryScan[]>([]);
  const [results, setResults] = useState<DiscoveryResult[]>([]);
  const [selectedPrefix, setSelectedPrefix] = useState<string>('');
  const [filterPrefix, setFilterPrefix] = useState<string>('');
  const [filterClass, setFilterClass] = useState<string>('');
  const [filterIgnored, setFilterIgnored] = useState<'all' | 'yes' | 'no'>('no');
  const [error, setError] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [scanning, setScanning] = useState(false);
  const { subscribe } = useEvents();

  const refresh = async () => {
    const [prefixesRes, scansRes, resultsRes] = await Promise.all([
      getPrefixes(),
      listDiscoveryScans({ limit: 25 }),
      listDiscoveryResults({
        prefix_id: filterPrefix || undefined,
        classification: (filterClass as DiscoveryClassification) || undefined,
        ignored: filterIgnored === 'all' ? undefined : filterIgnored === 'yes',
        limit: 200,
      }),
    ]);
    setPrefixes(prefixesRes.data || []);
    setScans(scansRes.data || []);
    setResults(resultsRes.data || []);
  };

  useEffect(() => {
    setLoading(true);
    refresh()
      .catch((e) => setError(String(e)))
      .finally(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filterPrefix, filterClass, filterIgnored]);

  useEffect(() => {
    const interval = setInterval(() => {
      refresh().catch(() => {});
    }, 30000); // Polling reduced

    const unsubscribe = subscribe((event) => {
      if (event.type.startsWith('discovery.')) {
        refresh();
      }
    });

    return () => {
      clearInterval(interval);
      unsubscribe();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filterPrefix, filterClass, filterIgnored, subscribe]);

  const prefixById = useMemo(() => {
    const m = new Map<string, Prefix>();
    prefixes.forEach((p) => m.set(p.id, p));
    return m;
  }, [prefixes]);

  const handleStartScan = async () => {
    if (!selectedPrefix) {
      setError('Select a prefix to scan.');
      return;
    }
    setError('');
    setScanning(true);
    try {
      await startDiscoveryScan(selectedPrefix);
      await refresh();
    } catch (e: any) {
      setError(e?.response?.data?.error || e?.message || 'Failed to start scan');
    } finally {
      setScanning(false);
    }
  };

  const handleAccept = async (r: DiscoveryResult) => {
    setError('');
    try {
      await acceptDiscoveryResult(r.id, {});
      await refresh();
    } catch (e: any) {
      setError(e?.response?.data?.error || 'Failed to accept');
    }
  };

  const handleIgnore = async (r: DiscoveryResult) => {
    setError('');
    try {
      await ignoreDiscoveryResult(r.id);
      await refresh();
    } catch (e: any) {
      setError(e?.response?.data?.error || 'Failed to ignore');
    }
  };

  const handleCreateMonitor = (r: DiscoveryResult) => {
    const ports = r.open_ports || [];
    let monitor_type: 'http' | 'tcp' | 'dns' | 'tls' | 'postgres' = 'tcp';
    let target = r.address;

    if (ports.includes(5432)) {
      monitor_type = 'postgres';
      target = `postgres://user:pass@${r.address}:5432/dbname`;
    } else if (ports.includes(443)) {
      monitor_type = 'tls';
      target = r.address;
    } else if (ports.includes(53) && (r.reverse_dns || r.hostname)) {
      monitor_type = 'dns';
      target = r.reverse_dns || r.hostname || r.address;
    } else if (ports.includes(80)) {
      monitor_type = 'http';
      target = `http://${r.address}`;
    } else if (ports.length > 0) {
      monitor_type = 'tcp';
      target = `${r.address}:${ports[0]}`;
    }

    const params = new URLSearchParams({
      type: monitor_type,
      target,
      name: r.reverse_dns || r.hostname || r.address,
    });
    navigate(`/monitors?${params.toString()}`);
  };

  const newHosts = useMemo(
    () => results.filter((r) => r.classification === 'new' && !r.ignored).slice(0, 10),
    [results],
  );

  const conflictsAndStale = useMemo(
    () => results.filter((r) => r.classification === 'duplicate' || r.classification === 'stale').slice(0, 20),
    [results],
  );

  if (loading) return <div className="p-4 text-signal-amber">Loading discovery data...</div>;

  return (
    <div className="space-y-6">
      <div className="panel">
        <h2 className="text-xl text-signal-green mb-4">Manual Scan</h2>
        <div className="flex flex-wrap items-center gap-2">
          <select
            value={selectedPrefix}
            onChange={(e) => setSelectedPrefix(e.target.value)}
            className="bg-surface border border-surface text-text-main px-3 py-2 rounded text-sm"
          >
            <option value="">— select prefix —</option>
            {prefixes.map((p) => (
              <option key={p.id} value={p.id}>
                {p.prefix} {p.description ? `(${p.description})` : ''}
              </option>
            ))}
          </select>
          <button
            onClick={handleStartScan}
            disabled={scanning || !selectedPrefix}
            className="bg-signal-green/10 border border-signal-green text-signal-green px-4 py-2 rounded text-sm hover:bg-signal-green/20 disabled:opacity-50"
          >
            {scanning ? 'STARTING…' : 'START SCAN'}
          </button>
          {error && <span className="text-signal-red text-sm">{error}</span>}
        </div>
      </div>

      <div className="panel">
        <h2 className="text-xl text-signal-green mb-4">Recent Scans</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse text-sm">
            <thead>
              <tr className="border-b border-surface">
                <th className="p-2">Status</th>
                <th className="p-2">Prefix</th>
                <th className="p-2">Started</th>
                <th className="p-2">Completed</th>
                <th className="p-2">Duration</th>
                <th className="p-2">Error</th>
              </tr>
            </thead>
            <tbody>
              {scans.length === 0 ? (
                <tr>
                  <td colSpan={6} className="p-2 text-text-muted">
                    No scans yet
                  </td>
                </tr>
              ) : (
                scans.map((s) => (
                  <tr key={s.id} className="border-b border-surface">
                    <td className="p-2">
                      <span
                        className={
                          s.status === 'completed'
                            ? 'text-signal-green'
                            : s.status === 'failed'
                            ? 'text-signal-red'
                            : 'text-signal-amber'
                        }
                      >
                        [{s.status.toUpperCase()}]
                      </span>
                    </td>
                    <td className="p-2 text-signal-green">{prefixById.get(s.prefix_id)?.prefix || s.prefix_id}</td>
                    <td className="p-2 text-text-muted">{s.started_at ? new Date(s.started_at).toLocaleString() : '-'}</td>
                    <td className="p-2 text-text-muted">{s.completed_at ? new Date(s.completed_at).toLocaleString() : '-'}</td>
                    <td className="p-2 text-text-muted">{formatDuration(s)}</td>
                    <td className="p-2 text-signal-red">{s.error || ''}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      <div className="panel">
        <h2 className="text-xl text-signal-green mb-4">Results</h2>
        <div className="flex flex-wrap gap-2 mb-4">
          <select
            value={filterPrefix}
            onChange={(e) => setFilterPrefix(e.target.value)}
            className="bg-surface border border-surface text-text-main px-3 py-2 rounded text-sm"
          >
            <option value="">all prefixes</option>
            {prefixes.map((p) => (
              <option key={p.id} value={p.id}>
                {p.prefix}
              </option>
            ))}
          </select>
          <select
            value={filterClass}
            onChange={(e) => setFilterClass(e.target.value)}
            className="bg-surface border border-surface text-text-main px-3 py-2 rounded text-sm"
          >
            <option value="">all classifications</option>
            {CLASSIFICATIONS.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
          <select
            value={filterIgnored}
            onChange={(e) => setFilterIgnored(e.target.value as 'all' | 'yes' | 'no')}
            className="bg-surface border border-surface text-text-main px-3 py-2 rounded text-sm"
          >
            <option value="no">not ignored</option>
            <option value="yes">ignored only</option>
            <option value="all">all</option>
          </select>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse text-sm font-mono">
            <thead>
              <tr className="border-b border-surface">
                <th className="p-2">Class</th>
                <th className="p-2">Address</th>
                <th className="p-2">Reverse DNS</th>
                <th className="p-2">Open Ports</th>
                <th className="p-2">Latency</th>
                <th className="p-2">Seen</th>
                <th className="p-2">Actions</th>
              </tr>
            </thead>
            <tbody>
              {results.length === 0 ? (
                <tr>
                  <td colSpan={7} className="p-2 text-text-muted">
                    No results
                  </td>
                </tr>
              ) : (
                results.map((r) => (
                  <tr key={r.id} className="border-b border-surface">
                    <td className={`p-2 ${classificationStyle[r.classification]}`}>{classificationTag(r.classification)}</td>
                    <td className="p-2 text-signal-green">{r.address}</td>
                    <td className="p-2 text-text-muted">{r.reverse_dns || '-'}</td>
                    <td className="p-2 text-text-muted">{(r.open_ports || []).join(', ') || '-'}</td>
                    <td className="p-2 text-text-muted">{r.latency_ms ? `${r.latency_ms}ms` : '-'}</td>
                    <td className="p-2 text-text-muted">{new Date(r.seen_at).toLocaleString()}</td>
                    <td className="p-2 space-x-2">
                      {r.classification === 'new' && !r.accepted_at && !r.ignored && (
                        <button onClick={() => handleAccept(r)} className="text-signal-green hover:underline">
                          accept
                        </button>
                      )}
                      {!r.ignored && (
                        <button onClick={() => handleIgnore(r)} className="text-signal-amber hover:underline">
                          ignore
                        </button>
                      )}
                      {r.accepted_at && r.created_ip_address_id && (
                        <span className="text-signal-green">→ in IPAM</span>
                      )}
                      <button onClick={() => handleCreateMonitor(r)} className="text-signal-green hover:underline">
                        monitor
                      </button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="panel">
          <h2 className="text-xl text-signal-amber mb-4">New Hosts</h2>
          {newHosts.length === 0 ? (
            <div className="text-text-muted text-sm">No new hosts.</div>
          ) : (
            <ul className="text-sm font-mono space-y-1">
              {newHosts.map((r) => (
                <li key={r.id} className="flex justify-between">
                  <span>
                    <span className="text-signal-amber">[NEW]</span> {r.address}
                  </span>
                  <span className="text-text-muted">{(r.open_ports || []).slice(0, 3).join(',')}</span>
                </li>
              ))}
            </ul>
          )}
        </div>

        <div className="panel">
          <h2 className="text-xl text-signal-red mb-4">Conflicts &amp; Stale</h2>
          {conflictsAndStale.length === 0 ? (
            <div className="text-text-muted text-sm">No conflicts or stale hosts.</div>
          ) : (
            <ul className="text-sm font-mono space-y-1">
              {conflictsAndStale.map((r) => (
                <li key={r.id}>
                  <span className={classificationStyle[r.classification]}>{classificationTag(r.classification)}</span>{' '}
                  {r.address}
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  );
}
