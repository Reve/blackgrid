import { useEffect, useState } from 'react';
import type { Prefix, IPAddress, DiscoveryScan } from '../api/client';
import {
  getPrefixes,
  getIPAddresses,
  startPrefixScan,
  listDiscoveryScans,
  updatePrefixScanConfig,
} from '../api/client';

export default function IPAM() {
  const [prefixes, setPrefixes] = useState<Prefix[]>([]);
  const [ips, setIps] = useState<IPAddress[]>([]);
  const [scansByPrefix, setScansByPrefix] = useState<Record<string, DiscoveryScan>>({});
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState<string>('');
  const [error, setError] = useState<string>('');

  const loadAll = async () => {
    const [prefixesRes, ipsRes, scansRes] = await Promise.all([
      getPrefixes(),
      getIPAddresses(),
      listDiscoveryScans({ limit: 200 }),
    ]);
    setPrefixes(prefixesRes.data || []);
    setIps(ipsRes.data || []);
    const latest: Record<string, DiscoveryScan> = {};
    for (const s of scansRes.data || []) {
      if (!latest[s.prefix_id]) latest[s.prefix_id] = s;
    }
    setScansByPrefix(latest);
  };

  useEffect(() => {
    loadAll()
      .catch((e) => setError(String(e)))
      .finally(() => setLoading(false));
  }, []);

  const handleScanNow = async (prefixId: string) => {
    setError('');
    setBusy(prefixId);
    try {
      await startPrefixScan(prefixId);
      await loadAll();
    } catch (e: any) {
      setError(e?.response?.data?.error || e?.message || 'Failed to start scan');
    } finally {
      setBusy('');
    }
  };

  const handleToggleScanEnabled = async (p: Prefix) => {
    setError('');
    try {
      await updatePrefixScanConfig(p.id, {
        scan_enabled: !p.scan_enabled,
        scan_interval_seconds: p.scan_interval_seconds || 3600,
      });
      await loadAll();
    } catch (e: any) {
      setError(e?.response?.data?.error || 'Failed to update scan config');
    }
  };

  const handleIntervalChange = async (p: Prefix, value: number) => {
    if (value < 60) {
      setError('Interval must be >= 60 seconds');
      return;
    }
    setError('');
    try {
      await updatePrefixScanConfig(p.id, {
        scan_enabled: !!p.scan_enabled,
        scan_interval_seconds: value,
      });
      await loadAll();
    } catch (e: any) {
      setError(e?.response?.data?.error || 'Failed to update scan config');
    }
  };

  if (loading) return <div className="p-4 text-signal-amber">Loading IPAM data...</div>;

  return (
    <div className="space-y-6">
      {error && <div className="panel border border-signal-red text-signal-red text-sm">{error}</div>}
      <div className="panel">
        <h2 className="text-xl text-signal-green mb-4">Prefixes</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse text-sm">
            <thead>
              <tr className="border-b border-surface">
                <th className="p-2">Prefix</th>
                <th className="p-2">Description</th>
                <th className="p-2">Scan</th>
                <th className="p-2">Interval (s)</th>
                <th className="p-2">Last Scan</th>
                <th className="p-2">Actions</th>
              </tr>
            </thead>
            <tbody>
              {prefixes.length === 0 ? (
                <tr>
                  <td colSpan={6} className="p-2 text-text-muted">
                    No prefixes found
                  </td>
                </tr>
              ) : (
                prefixes.map((p) => {
                  const last = scansByPrefix[p.id];
                  return (
                    <tr key={p.id} className="border-b border-surface">
                      <td className="p-2 text-signal-green">{p.prefix}</td>
                      <td className="p-2 text-text-muted">{p.description}</td>
                      <td className="p-2">
                        <label className="inline-flex items-center gap-2">
                          <input
                            type="checkbox"
                            checked={!!p.scan_enabled}
                            onChange={() => handleToggleScanEnabled(p)}
                          />
                          <span className={p.scan_enabled ? 'text-signal-green' : 'text-text-muted'}>
                            {p.scan_enabled ? 'enabled' : 'disabled'}
                          </span>
                        </label>
                      </td>
                      <td className="p-2">
                        <input
                          type="number"
                          min={60}
                          defaultValue={p.scan_interval_seconds || 3600}
                          onBlur={(e) => {
                            const v = Number(e.target.value);
                            if (v !== p.scan_interval_seconds) handleIntervalChange(p, v);
                          }}
                          className="w-24 bg-surface border border-surface text-text-main px-2 py-1 rounded"
                        />
                      </td>
                      <td className="p-2 text-text-muted">
                        {last ? (
                          <span>
                            <span
                              className={
                                last.status === 'completed'
                                  ? 'text-signal-green'
                                  : last.status === 'failed'
                                  ? 'text-signal-red'
                                  : 'text-signal-amber'
                              }
                            >
                              [{last.status}]
                            </span>{' '}
                            {last.completed_at
                              ? new Date(last.completed_at).toLocaleString()
                              : last.started_at
                              ? new Date(last.started_at).toLocaleString()
                              : ''}
                          </span>
                        ) : (
                          'never'
                        )}
                      </td>
                      <td className="p-2">
                        <button
                          onClick={() => handleScanNow(p.id)}
                          disabled={busy === p.id}
                          className="text-signal-green hover:underline disabled:opacity-50"
                        >
                          {busy === p.id ? 'starting…' : 'scan now'}
                        </button>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      </div>

      <div className="panel">
        <h2 className="text-xl text-signal-green mb-4">IP Addresses</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse text-sm">
            <thead>
              <tr className="border-b border-surface">
                <th className="p-2">IP Address</th>
                <th className="p-2">Status</th>
                <th className="p-2">Description</th>
                <th className="p-2">Last Seen</th>
              </tr>
            </thead>
            <tbody>
              {ips.length === 0 ? (
                <tr>
                  <td colSpan={4} className="p-2 text-text-muted">
                    No IP addresses found
                  </td>
                </tr>
              ) : (
                ips.map((ip) => (
                  <tr key={ip.id} className="border-b border-surface">
                    <td className="p-2 text-signal-green">{ip.ip_address}</td>
                    <td className="p-2">
                      <span
                        className={`px-2 py-1 rounded text-xs ${
                          ip.status === 'conflict'
                            ? 'bg-signal-red/20 text-signal-red'
                            : ip.status === 'stale'
                            ? 'bg-signal-amber/20 text-signal-amber'
                            : ip.status === 'active' || ip.status === 'discovered'
                            ? 'bg-signal-green/20 text-signal-green'
                            : 'bg-surface text-text-muted'
                        }`}
                      >
                        {ip.status}
                      </span>
                    </td>
                    <td className="p-2 text-text-muted">{ip.description}</td>
                    <td className="p-2 text-text-muted">
                      {ip.last_seen_at ? new Date(ip.last_seen_at).toLocaleString() : '-'}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
