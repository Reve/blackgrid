import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  acceptDiscoveryResult,
  getPrefixes,
  ignoreDiscoveryResult,
  listDiscoveryResults,
  listDiscoveryScans,
  startDiscoveryScan,
  getDiscoveryDiagnostics,
  probeDiscoveryHost,
  type DiscoveryClassification,
  type DiscoveryResult,
  type DiscoveryScan,
  type Prefix,
  type DiscoveryDiagnostics,
  type DiscoveryProbeResponse,
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

function describeError(e: unknown, fallback: string): string {
  if (!e) return fallback;
  if (typeof e === 'string') return e;
  if (typeof e === 'object') {
    const err = e as {
      message?: string;
      response?: { data?: { error?: { message?: string } } };
    };
    return err.message || err.response?.data?.error?.message || fallback;
  }
  return fallback;
}

function mergeScans(current: DiscoveryScan[], incoming: DiscoveryScan[]) {
  const byId = new Map<string, DiscoveryScan>();
  for (const scan of current) byId.set(scan.id, scan);
  for (const scan of incoming) byId.set(scan.id, scan);
  return [...byId.values()]
    .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
    .slice(0, 25);
}

function portsList(value: unknown): number[] {
  if (Array.isArray(value)) {
    return value.filter((port): port is number => typeof port === 'number');
  }
  if (typeof value === 'string') {
    try {
      const parsed = JSON.parse(value);
      if (Array.isArray(parsed)) {
        return parsed.filter((port): port is number => typeof port === 'number');
      }
    } catch {
      return [];
    }
  }
  return [];
}

function formatPorts(value: unknown, limit?: number): string {
  const ports = portsList(value);
  const visible = limit ? ports.slice(0, limit) : ports;
  return visible.join(', ') || '-';
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
  const [filterPorts, setFilterPorts] = useState<string>('');
  const [error, setError] = useState<string>('');
  const [scanStatus, setScanStatus] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [scanning, setScanning] = useState(false);
  const [diagnostics, setDiagnostics] = useState<DiscoveryDiagnostics | null>(null);
  const [probeAddr, setProbeAddr] = useState('');
  const [probePorts, setProbePorts] = useState('');
  const [probing, setProbing] = useState(false);
  const [probeResult, setProbeResult] = useState<DiscoveryProbeResponse | null>(null);
  const [probeError, setProbeError] = useState('');
  const { subscribe } = useEvents();

  const refresh = async () => {
    const [prefixesRes, scansRes, resultsRes] = await Promise.all([
      getPrefixes(),
      listDiscoveryScans({ limit: 25 }),
      listDiscoveryResults({
        prefix_id: filterPrefix || undefined,
        classification: (filterClass as DiscoveryClassification) || undefined,
        ignored: filterIgnored === 'all' ? undefined : filterIgnored === 'yes',
        ports: filterPorts.trim() || undefined,
        limit: 200,
      }),
    ]);
    const nextPrefixes = prefixesRes.data || [];
    const nextScans = scansRes.data || [];
    const nextResults = resultsRes.data || [];

    setPrefixes(nextPrefixes);
    setScans((current) => mergeScans(current, nextScans));
    setResults(nextResults);
    return {
      prefixes: nextPrefixes,
      scans: nextScans,
      results: nextResults,
    };
  };

  useEffect(() => {
    const timeout = window.setTimeout(() => {
      refresh()
        .catch((e) => setError(String(e)))
        .finally(() => setLoading(false));
    }, 0);
    return () => window.clearTimeout(timeout);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filterPrefix, filterClass, filterIgnored, filterPorts]);

  useEffect(() => {
    getDiscoveryDiagnostics()
      .then((r) => setDiagnostics(r.data))
      .catch(() => setDiagnostics(null));
  }, []);

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
  }, [filterPrefix, filterClass, filterIgnored, filterPorts, subscribe]);

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
    setScanStatus('');
    setScanning(true);
    try {
      const started = (await startDiscoveryScan(selectedPrefix)).data;
      const prefixLabel = prefixById.get(selectedPrefix)?.prefix || selectedPrefix;
      setScans((current) => [started, ...current.filter((s) => s.id !== started.id)]);
      setScanStatus(`Scan queued for ${prefixLabel}.`);

      const refreshed = await refresh();
      const updated = refreshed.scans.find((s) => s.id === started.id) || started;
      const loadedResults = refreshed.results.filter((r) => r.scan_id === started.id).length;
      const suffix =
        updated.status === 'completed'
          ? ` ${loadedResults} result${loadedResults === 1 ? '' : 's'} loaded.`
          : '';
      setScanStatus(`Scan ${updated.status} for ${prefixLabel}.${suffix}`);
    } catch (e: unknown) {
      setError(describeError(e, 'Failed to start scan'));
    } finally {
      setScanning(false);
    }
  };

  const handleAccept = async (r: DiscoveryResult) => {
    setError('');
    try {
      await acceptDiscoveryResult(r.id, {});
      await refresh();
    } catch (e: unknown) {
      setError(describeError(e, 'Failed to accept'));
    }
  };

  const handleIgnore = async (r: DiscoveryResult) => {
    setError('');
    try {
      await ignoreDiscoveryResult(r.id);
      await refresh();
    } catch (e: unknown) {
      setError(describeError(e, 'Failed to ignore'));
    }
  };

  const handleCreateMonitor = (r: DiscoveryResult) => {
    const ports = portsList(r.open_ports);
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

  const handleProbe = async () => {
    setProbeError('');
    setProbeResult(null);
    if (!probeAddr.trim()) {
      setProbeError('Address is required');
      return;
    }
    let ports: number[] | undefined;
    if (probePorts.trim()) {
      ports = probePorts
        .split(',')
        .map((s) => parseInt(s.trim(), 10))
        .filter((n) => Number.isFinite(n) && n > 0 && n <= 65535);
      if (ports.length === 0) {
        setProbeError('No valid ports in list');
        return;
      }
    }
    setProbing(true);
    try {
      const r = await probeDiscoveryHost({ address: probeAddr.trim(), ports });
      setProbeResult(r.data);
    } catch (e: unknown) {
      setProbeError(describeError(e, 'Probe failed'));
    } finally {
      setProbing(false);
    }
  };

  const lastCompletedScan = useMemo(
    () => scans.find((s) => s.status === 'completed') || null,
    [scans],
  );

  const lastScanResults = useMemo(() => {
    if (!lastCompletedScan) return [] as DiscoveryResult[];
    return results.filter((r) => r.scan_id === lastCompletedScan.id);
  }, [results, lastCompletedScan]);

  const lastScanCounts = useMemo(() => {
    const counts: Record<string, number> = {
      new: 0,
      known: 0,
      changed: 0,
      stale: 0,
      conflict: 0,
    };
    for (const r of lastScanResults) {
      if (r.classification === 'duplicate') counts.conflict++;
      else if (counts[r.classification] !== undefined) counts[r.classification]++;
    }
    return counts;
  }, [lastScanResults]);

  const showEmptyScanHint =
    lastCompletedScan && lastScanResults.length === 0 && !filterPrefix && !filterClass;

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
      {/* Diagnostics */}
      <div className="panel">
        <h2 className="text-xl text-signal-green mb-4">Discovery Diagnostics</h2>
        {diagnostics ? (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-sm font-mono">
            <div>
              <span className="text-text-muted">Default Ports: </span>
              <span className="text-signal-green">
                {diagnostics.default_ports.join(', ')}
              </span>
            </div>
            <div>
              <span className="text-text-muted">TCP Timeout: </span>
              <span className="text-signal-green">{diagnostics.tcp_timeout_ms} ms</span>
            </div>
            <div>
              <span className="text-text-muted">Workers: </span>
              <span className="text-signal-green">{diagnostics.worker_count}</span>
            </div>
            <div>
              <span className="text-text-muted">Ping Supported: </span>
              <span
                className={
                  diagnostics.ping_supported ? 'text-signal-green' : 'text-signal-amber'
                }
              >
                {diagnostics.ping_supported ? 'yes' : 'no'}
              </span>
            </div>
            <div>
              <span className="text-text-muted">Inside Container: </span>
              <span className="text-signal-green">
                {diagnostics.runtime.inside_container ? 'yes' : 'no'}
              </span>
            </div>
            <div>
              <span className="text-text-muted">Hostname: </span>
              <span className="text-signal-green">{diagnostics.runtime.hostname}</span>
            </div>
          </div>
        ) : (
          <div className="text-text-muted text-sm">
            Diagnostics unavailable (operator/admin role required).
          </div>
        )}
        <div className="mt-4 text-xs text-text-muted italic">
          Discovery currently detects hosts by TCP open ports unless ping support is
          enabled.
        </div>
      </div>

      {/* Probe Host */}
      <div className="panel">
        <h2 className="text-xl text-signal-green mb-4">Probe Host</h2>
        <div className="flex flex-wrap items-end gap-2">
          <label className="flex flex-col text-xs uppercase tracking-[0.12em] text-text-muted">
            Address
            <input
              placeholder="10.10.13.1"
              className="mt-1 bg-surface border border-surface text-text-main px-3 py-2 rounded text-sm w-48"
              value={probeAddr}
              onChange={(e) => setProbeAddr(e.target.value)}
            />
          </label>
          <label className="flex flex-col text-xs uppercase tracking-[0.12em] text-text-muted">
            Ports (comma-separated, optional)
            <input
              placeholder="22,80,443"
              className="mt-1 bg-surface border border-surface text-text-main px-3 py-2 rounded text-sm w-64"
              value={probePorts}
              onChange={(e) => setProbePorts(e.target.value)}
            />
          </label>
          <button
            onClick={handleProbe}
            disabled={probing}
            className="bg-signal-green/10 border border-signal-green text-signal-green px-4 py-2 rounded text-sm hover:bg-signal-green/20 disabled:opacity-50"
          >
            {probing ? 'PROBING…' : 'PROBE'}
          </button>
        </div>
        {probeError && <div className="text-signal-red text-sm mt-2">{probeError}</div>}
        {probeResult && (
          <div className="mt-4 text-sm font-mono space-y-1">
            <div>
              <span className="text-text-muted">seen: </span>
              <span
                className={probeResult.seen ? 'text-signal-green' : 'text-signal-amber'}
              >
                {probeResult.seen ? 'yes' : 'no'}
              </span>
            </div>
            <div>
              <span className="text-text-muted">open ports: </span>
              <span className="text-signal-green">
                {probeResult.open_ports.length === 0
                  ? '-'
                  : probeResult.open_ports.join(', ')}
              </span>
            </div>
            <div>
              <span className="text-text-muted">latency: </span>
              <span className="text-signal-green">
                {probeResult.latency_ms ? `${probeResult.latency_ms}ms` : '-'}
              </span>
            </div>
            <div>
              <span className="text-text-muted">reverse dns: </span>
              <span className="text-signal-green">{probeResult.reverse_dns || '-'}</span>
            </div>
          </div>
        )}
      </div>

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
          {scanStatus && !error && (
            <span className="text-signal-green text-sm">{scanStatus}</span>
          )}
        </div>
      </div>

      {showEmptyScanHint && (
        <div className="panel border-l-2 border-l-signal-amber bg-signal-amber/5">
          <h3 className="text-signal-amber uppercase tracking-[0.14em] text-sm mb-2">
            ■ No hosts detected in last scan
          </h3>
          <p className="text-sm text-text-muted">
            Blackgrid discovery uses TCP probes against configured ports. The subnet may
            be reachable but hosts with closed/firewalled ports will not appear. Use
            Probe Host to test a known IP/port, or configure{' '}
            <code className="text-signal-green">DISCOVERY_DEFAULT_PORTS</code>.
          </p>
        </div>
      )}

      <div className="panel">
        <h2 className="text-xl text-signal-green mb-4">Recent Scans</h2>
        {lastCompletedScan && (
          <div className="mb-3 text-xs font-mono text-text-muted">
            Latest completed:{' '}
            <span className="text-signal-green">new {lastScanCounts.new}</span>{' '}
            <span className="text-signal-green">known {lastScanCounts.known}</span>{' '}
            <span className="text-signal-amber">changed {lastScanCounts.changed}</span>{' '}
            <span className="text-text-muted">stale {lastScanCounts.stale}</span>{' '}
            <span className="text-signal-red">conflict {lastScanCounts.conflict}</span>
          </div>
        )}
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
          <input
            value={filterPorts}
            onChange={(e) => setFilterPorts(e.target.value)}
            placeholder="ports: 22,80,443"
            className="bg-surface border border-surface text-text-main px-3 py-2 rounded text-sm w-44"
          />
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
                    <td className="p-2 text-text-muted">{formatPorts(r.open_ports)}</td>
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
                  <span className="text-text-muted">{formatPorts(r.open_ports, 3)}</span>
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
