import { useEffect, useState } from 'react';
import type {
  Prefix,
  IPAddress,
  DiscoveryScan,
  Site,
  CreateSiteInput,
  CreatePrefixInput,
  CreateIPAddressInput,
} from '../api/client';
import {
  getSites,
  createSite,
  updateSite,
  deleteSite,
  getPrefixes,
  createPrefix,
  updatePrefix,
  deletePrefix,
  getIPAddresses,
  createIPAddress,
  updateIPAddress,
  deleteIPAddress,
  reserveIPAddress,
  assignIPAddress,
  releaseIPAddress,
  startPrefixScan,
  listDiscoveryScans,
  updatePrefixScanConfig,
  type ApiErrorDetail,
} from '../api/client';
import { Loading, ErrorState } from '../components/UI';
import { useToast } from '../context/ToastContext';
import { useAuth } from '../context/AuthContext';

const IP_STATUSES = [
  'available',
  'reserved',
  'assigned',
  'dhcp',
  'discovered',
  'conflict',
  'deprecated',
  'active',
];

interface SiteFormState {
  open: boolean;
  editing?: Site;
  name: string;
  description: string;
}
const blankSiteForm: SiteFormState = { open: false, name: '', description: '' };

interface PrefixFormState {
  open: boolean;
  editing?: Prefix;
  site_id: string;
  vlan_id: string;
  prefix: string;
  description: string;
}
const blankPrefixForm: PrefixFormState = {
  open: false,
  site_id: '',
  vlan_id: '',
  prefix: '',
  description: '',
};

interface IPFormState {
  open: boolean;
  editing?: IPAddress;
  prefix_id: string;
  ip_address: string;
  status: string;
  description: string;
}
const blankIPForm: IPFormState = {
  open: false,
  prefix_id: '',
  ip_address: '',
  status: 'available',
  description: '',
};

function describeError(e: any): string {
  if (!e) return 'Unknown error';
  if (typeof e === 'string') return e;
  return e.message || 'Request failed';
}

export default function IPAM() {
  const { isOperator } = useAuth();
  const [sites, setSites] = useState<Site[]>([]);
  const [prefixes, setPrefixes] = useState<Prefix[]>([]);
  const [ips, setIps] = useState<IPAddress[]>([]);
  const [scansByPrefix, setScansByPrefix] = useState<Record<string, DiscoveryScan>>({});
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState<string>('');
  const [error, setError] = useState<ApiErrorDetail | null>(null);
  const { success, error: toastError } = useToast();

  const [siteForm, setSiteForm] = useState<SiteFormState>(blankSiteForm);
  const [prefixForm, setPrefixForm] = useState<PrefixFormState>(blankPrefixForm);
  const [ipForm, setIpForm] = useState<IPFormState>(blankIPForm);

  const loadAll = async () => {
    try {
      const [sitesRes, prefixesRes, ipsRes, scansRes] = await Promise.all([
        getSites(),
        getPrefixes(),
        getIPAddresses(),
        listDiscoveryScans({ limit: 200 }),
      ]);
      setSites(sitesRes.data || []);
      setPrefixes(prefixesRes.data || []);
      setIps(ipsRes.data || []);
      const latest: Record<string, DiscoveryScan> = {};
      for (const s of scansRes.data || []) {
        if (!latest[s.prefix_id]) latest[s.prefix_id] = s;
      }
      setScansByPrefix(latest);
      setError(null);
    } catch (e: any) {
      setError(e);
    }
  };

  useEffect(() => {
    loadAll().finally(() => setLoading(false));
  }, []);

  const handleScanNow = async (prefixId: string) => {
    setBusy(prefixId);
    try {
      await startPrefixScan(prefixId);
      success('Scan queued successfully');
      await loadAll();
    } catch (e: any) {
      toastError(describeError(e));
    } finally {
      setBusy('');
    }
  };

  const handleToggleScanEnabled = async (p: Prefix) => {
    setError(null);
    try {
      await updatePrefixScanConfig(p.id, {
        scan_enabled: !p.scan_enabled,
        scan_interval_seconds: p.scan_interval_seconds || 3600,
      });
      await loadAll();
    } catch (e: any) {
      toastError(describeError(e));
    }
  };

  const handleIntervalChange = async (p: Prefix, value: number) => {
    if (value < 60) {
      toastError('Interval must be >= 60 seconds');
      return;
    }
    try {
      await updatePrefixScanConfig(p.id, {
        scan_enabled: !!p.scan_enabled,
        scan_interval_seconds: value,
      });
      success('Scan interval updated');
      await loadAll();
    } catch (e: any) {
      toastError(describeError(e));
    }
  };

  // ---- Sites ----
  const openCreateSite = () => setSiteForm({ ...blankSiteForm, open: true });
  const openEditSite = (s: Site) =>
    setSiteForm({ open: true, editing: s, name: s.name, description: s.description || '' });
  const closeSite = () => setSiteForm(blankSiteForm);

  const submitSite = async () => {
    if (!siteForm.name.trim()) {
      toastError('Site name is required');
      return;
    }
    const data: CreateSiteInput = {
      name: siteForm.name.trim(),
      description: siteForm.description || null,
    };
    try {
      if (siteForm.editing) {
        await updateSite(siteForm.editing.id, data);
        success('Site updated');
      } else {
        await createSite(data);
        success('Site created');
      }
      closeSite();
      await loadAll();
    } catch (e: any) {
      toastError(describeError(e));
    }
  };

  const handleDeleteSite = async (s: Site) => {
    if (!confirm(`Delete site "${s.name}"?`)) return;
    try {
      await deleteSite(s.id);
      success('Site deleted');
      await loadAll();
    } catch (e: any) {
      toastError(describeError(e));
    }
  };

  // ---- Prefixes ----
  const openCreatePrefix = () => setPrefixForm({ ...blankPrefixForm, open: true });
  const openEditPrefix = (p: Prefix) =>
    setPrefixForm({
      open: true,
      editing: p,
      site_id: p.site_id || '',
      vlan_id: p.vlan_id || '',
      prefix: p.prefix,
      description: p.description || '',
    });
  const closePrefix = () => setPrefixForm(blankPrefixForm);

  const submitPrefix = async () => {
    if (!prefixForm.prefix.trim()) {
      toastError('Prefix (CIDR) is required');
      return;
    }
    if (!prefixForm.site_id) {
      toastError('Site is required');
      return;
    }
    const data: CreatePrefixInput = {
      site_id: prefixForm.site_id,
      vlan_id: prefixForm.vlan_id || null,
      prefix: prefixForm.prefix.trim(),
      description: prefixForm.description || null,
    };
    try {
      if (prefixForm.editing) {
        await updatePrefix(prefixForm.editing.id, data);
        success('Prefix updated');
      } else {
        await createPrefix(data);
        success('Prefix created');
      }
      closePrefix();
      await loadAll();
    } catch (e: any) {
      toastError(describeError(e));
    }
  };

  const handleDeletePrefix = async (p: Prefix) => {
    if (!confirm(`Delete prefix "${p.prefix}"?`)) return;
    try {
      await deletePrefix(p.id);
      success('Prefix deleted');
      await loadAll();
    } catch (e: any) {
      toastError(describeError(e));
    }
  };

  // ---- IPs ----
  const openCreateIP = () => {
    const defaultPrefix = prefixes[0]?.id || '';
    setIpForm({ ...blankIPForm, open: true, prefix_id: defaultPrefix });
  };
  const openEditIP = (ip: IPAddress) =>
    setIpForm({
      open: true,
      editing: ip,
      prefix_id: ip.prefix_id,
      ip_address: ip.ip_address,
      status: ip.status || 'available',
      description: ip.description || '',
    });
  const closeIP = () => setIpForm(blankIPForm);

  const submitIP = async () => {
    if (!ipForm.prefix_id) {
      toastError('Prefix is required');
      return;
    }
    if (!ipForm.ip_address.trim()) {
      toastError('IP address is required');
      return;
    }
    const data: CreateIPAddressInput = {
      prefix_id: ipForm.prefix_id,
      ip_address: ipForm.ip_address.trim(),
      status: ipForm.status || 'available',
      description: ipForm.description || null,
    };
    try {
      if (ipForm.editing) {
        await updateIPAddress(ipForm.editing.id, data);
        success('IP address updated');
      } else {
        await createIPAddress(data);
        success('IP address created');
      }
      closeIP();
      await loadAll();
    } catch (e: any) {
      toastError(describeError(e));
    }
  };

  const handleDeleteIP = async (ip: IPAddress) => {
    if (!confirm(`Delete IP ${ip.ip_address}?`)) return;
    try {
      await deleteIPAddress(ip.id);
      success('IP deleted');
      await loadAll();
    } catch (e: any) {
      toastError(describeError(e));
    }
  };

  const ipAction = async (ip: IPAddress, action: 'reserve' | 'assign' | 'release') => {
    try {
      if (action === 'reserve') await reserveIPAddress(ip.id);
      else if (action === 'assign') await assignIPAddress(ip.id);
      else await releaseIPAddress(ip.id);
      success(`IP ${action}d`);
      await loadAll();
    } catch (e: any) {
      toastError(describeError(e));
    }
  };

  if (loading) return <Loading message="Accessing IPAM records..." />;
  if (error) return <ErrorState error={error} onRetry={loadAll} />;

  const siteName = (id?: string | null) => sites.find((s) => s.id === id)?.name || '-';

  return (
    <div className="space-y-6">
      {/* SITES */}
      <div className="panel">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl text-signal-green">Sites</h2>
          {isOperator && (
            <button
              onClick={openCreateSite}
              className="text-signal-green border border-signal-green px-3 py-1 text-xs uppercase tracking-[0.14em] hover:bg-signal-green/10"
            >
              + New Site
            </button>
          )}
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse text-sm">
            <thead>
              <tr className="border-b border-surface">
                <th className="p-2">Name</th>
                <th className="p-2">Description</th>
                <th className="p-2">Actions</th>
              </tr>
            </thead>
            <tbody>
              {sites.length === 0 ? (
                <tr>
                  <td colSpan={3} className="p-2 text-text-muted">
                    No sites yet. Create one to get started.
                  </td>
                </tr>
              ) : (
                sites.map((s) => (
                  <tr key={s.id} className="border-b border-surface">
                    <td className="p-2 text-signal-green">{s.name}</td>
                    <td className="p-2 text-text-muted">{s.description || ''}</td>
                    <td className="p-2 space-x-3">
                      {isOperator ? (
                        <>
                          <button
                            onClick={() => openEditSite(s)}
                            className="text-signal-green hover:underline"
                          >
                            edit
                          </button>
                          <button
                            onClick={() => handleDeleteSite(s)}
                            className="text-signal-red hover:underline"
                          >
                            delete
                          </button>
                        </>
                      ) : (
                        <span className="text-text-muted text-xs">read-only</span>
                      )}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {siteForm.open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4">
          <div className="panel max-w-md w-full">
            <h3 className="text-signal-green mb-4 uppercase tracking-[0.14em]">
              {siteForm.editing ? 'Edit Site' : 'New Site'}
            </h3>
            <div className="space-y-3">
              <label className="block text-xs uppercase tracking-[0.12em] text-text-muted">
                Name
                <input
                  className="mt-1 w-full bg-surface border border-surface text-text-main px-3 py-2 rounded"
                  value={siteForm.name}
                  onChange={(e) => setSiteForm((f) => ({ ...f, name: e.target.value }))}
                />
              </label>
              <label className="block text-xs uppercase tracking-[0.12em] text-text-muted">
                Description
                <input
                  className="mt-1 w-full bg-surface border border-surface text-text-main px-3 py-2 rounded"
                  value={siteForm.description}
                  onChange={(e) =>
                    setSiteForm((f) => ({ ...f, description: e.target.value }))
                  }
                />
              </label>
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button onClick={closeSite} className="terminal-button py-1 px-4 text-xs">
                Cancel
              </button>
              <button
                onClick={submitSite}
                className="border border-signal-green text-signal-green py-1 px-4 text-xs uppercase tracking-[0.14em] hover:bg-signal-green/10"
              >
                Save
              </button>
            </div>
          </div>
        </div>
      )}

      {/* PREFIXES */}
      <div className="panel">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl text-signal-green">Prefixes</h2>
          {isOperator && (
            <button
              onClick={openCreatePrefix}
              disabled={sites.length === 0}
              className="text-signal-green border border-signal-green px-3 py-1 text-xs uppercase tracking-[0.14em] hover:bg-signal-green/10 disabled:opacity-50"
              title={sites.length === 0 ? 'Create a site first' : ''}
            >
              + New Prefix
            </button>
          )}
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse text-sm">
            <thead>
              <tr className="border-b border-surface">
                <th className="p-2">Prefix</th>
                <th className="p-2">Site</th>
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
                  <td colSpan={7} className="p-2 text-text-muted">
                    No prefixes found
                  </td>
                </tr>
              ) : (
                prefixes.map((p) => {
                  const last = scansByPrefix[p.id];
                  return (
                    <tr key={p.id} className="border-b border-surface">
                      <td className="p-2 text-signal-green">{p.prefix}</td>
                      <td className="p-2 text-text-muted">{siteName(p.site_id)}</td>
                      <td className="p-2 text-text-muted">{p.description}</td>
                      <td className="p-2">
                        <label className="inline-flex items-center gap-2">
                          <input
                            type="checkbox"
                            checked={!!p.scan_enabled}
                            disabled={!isOperator}
                            onChange={() => handleToggleScanEnabled(p)}
                          />
                          <span
                            className={
                              p.scan_enabled ? 'text-signal-green' : 'text-text-muted'
                            }
                          >
                            {p.scan_enabled ? 'enabled' : 'disabled'}
                          </span>
                        </label>
                      </td>
                      <td className="p-2">
                        <input
                          type="number"
                          min={60}
                          defaultValue={p.scan_interval_seconds || 3600}
                          disabled={!isOperator}
                          onBlur={(e) => {
                            const v = Number(e.target.value);
                            if (v !== p.scan_interval_seconds) handleIntervalChange(p, v);
                          }}
                          className="w-24 bg-surface border border-surface text-text-main px-2 py-1 rounded disabled:opacity-50"
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
                      <td className="p-2 space-x-3">
                        {isOperator ? (
                          <>
                            <button
                              onClick={() => handleScanNow(p.id)}
                              disabled={busy === p.id}
                              className="text-signal-green hover:underline disabled:opacity-50"
                            >
                              {busy === p.id ? 'starting…' : 'scan'}
                            </button>
                            <button
                              onClick={() => openEditPrefix(p)}
                              className="text-signal-green hover:underline"
                            >
                              edit
                            </button>
                            <button
                              onClick={() => handleDeletePrefix(p)}
                              className="text-signal-red hover:underline"
                            >
                              delete
                            </button>
                          </>
                        ) : (
                          <span className="text-text-muted text-xs">read-only</span>
                        )}
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      </div>

      {prefixForm.open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4">
          <div className="panel max-w-md w-full">
            <h3 className="text-signal-green mb-4 uppercase tracking-[0.14em]">
              {prefixForm.editing ? 'Edit Prefix' : 'New Prefix'}
            </h3>
            <div className="space-y-3">
              <label className="block text-xs uppercase tracking-[0.12em] text-text-muted">
                Site
                <select
                  className="mt-1 w-full bg-surface border border-surface text-text-main px-3 py-2 rounded"
                  value={prefixForm.site_id}
                  onChange={(e) =>
                    setPrefixForm((f) => ({ ...f, site_id: e.target.value }))
                  }
                >
                  <option value="">— select site —</option>
                  {sites.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.name}
                    </option>
                  ))}
                </select>
              </label>
              <label className="block text-xs uppercase tracking-[0.12em] text-text-muted">
                Prefix (CIDR)
                <input
                  placeholder="10.10.13.0/24"
                  className="mt-1 w-full bg-surface border border-surface text-text-main px-3 py-2 rounded"
                  value={prefixForm.prefix}
                  onChange={(e) =>
                    setPrefixForm((f) => ({ ...f, prefix: e.target.value }))
                  }
                />
              </label>
              <label className="block text-xs uppercase tracking-[0.12em] text-text-muted">
                Description
                <input
                  className="mt-1 w-full bg-surface border border-surface text-text-main px-3 py-2 rounded"
                  value={prefixForm.description}
                  onChange={(e) =>
                    setPrefixForm((f) => ({ ...f, description: e.target.value }))
                  }
                />
              </label>
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button onClick={closePrefix} className="terminal-button py-1 px-4 text-xs">
                Cancel
              </button>
              <button
                onClick={submitPrefix}
                className="border border-signal-green text-signal-green py-1 px-4 text-xs uppercase tracking-[0.14em] hover:bg-signal-green/10"
              >
                Save
              </button>
            </div>
          </div>
        </div>
      )}

      {/* IP ADDRESSES */}
      <div className="panel">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl text-signal-green">IP Addresses</h2>
          {isOperator && (
            <button
              onClick={openCreateIP}
              disabled={prefixes.length === 0}
              className="text-signal-green border border-signal-green px-3 py-1 text-xs uppercase tracking-[0.14em] hover:bg-signal-green/10 disabled:opacity-50"
              title={prefixes.length === 0 ? 'Create a prefix first' : ''}
            >
              + New IP
            </button>
          )}
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse text-sm">
            <thead>
              <tr className="border-b border-surface">
                <th className="p-2">IP Address</th>
                <th className="p-2">Status</th>
                <th className="p-2">Description</th>
                <th className="p-2">Last Seen</th>
                <th className="p-2">Actions</th>
              </tr>
            </thead>
            <tbody>
              {ips.length === 0 ? (
                <tr>
                  <td colSpan={5} className="p-2 text-text-muted">
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
                      {ip.last_seen_at
                        ? new Date(ip.last_seen_at).toLocaleString()
                        : '-'}
                    </td>
                    <td className="p-2 space-x-2">
                      {isOperator ? (
                        <>
                          <button
                            onClick={() => ipAction(ip, 'reserve')}
                            className="text-signal-amber hover:underline"
                          >
                            reserve
                          </button>
                          <button
                            onClick={() => ipAction(ip, 'assign')}
                            className="text-signal-green hover:underline"
                          >
                            assign
                          </button>
                          <button
                            onClick={() => ipAction(ip, 'release')}
                            className="text-text-muted hover:underline"
                          >
                            release
                          </button>
                          <button
                            onClick={() => openEditIP(ip)}
                            className="text-signal-green hover:underline"
                          >
                            edit
                          </button>
                          <button
                            onClick={() => handleDeleteIP(ip)}
                            className="text-signal-red hover:underline"
                          >
                            delete
                          </button>
                        </>
                      ) : (
                        <span className="text-text-muted text-xs">read-only</span>
                      )}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {ipForm.open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4">
          <div className="panel max-w-md w-full">
            <h3 className="text-signal-green mb-4 uppercase tracking-[0.14em]">
              {ipForm.editing ? 'Edit IP Address' : 'New IP Address'}
            </h3>
            <div className="space-y-3">
              <label className="block text-xs uppercase tracking-[0.12em] text-text-muted">
                Prefix
                <select
                  className="mt-1 w-full bg-surface border border-surface text-text-main px-3 py-2 rounded"
                  value={ipForm.prefix_id}
                  onChange={(e) =>
                    setIpForm((f) => ({ ...f, prefix_id: e.target.value }))
                  }
                >
                  <option value="">— select prefix —</option>
                  {prefixes.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.prefix} {p.description ? `(${p.description})` : ''}
                    </option>
                  ))}
                </select>
              </label>
              <label className="block text-xs uppercase tracking-[0.12em] text-text-muted">
                IP Address
                <input
                  placeholder="10.10.13.5"
                  className="mt-1 w-full bg-surface border border-surface text-text-main px-3 py-2 rounded"
                  value={ipForm.ip_address}
                  onChange={(e) =>
                    setIpForm((f) => ({ ...f, ip_address: e.target.value }))
                  }
                />
              </label>
              <label className="block text-xs uppercase tracking-[0.12em] text-text-muted">
                Status
                <select
                  className="mt-1 w-full bg-surface border border-surface text-text-main px-3 py-2 rounded"
                  value={ipForm.status}
                  onChange={(e) => setIpForm((f) => ({ ...f, status: e.target.value }))}
                >
                  {IP_STATUSES.map((s) => (
                    <option key={s} value={s}>
                      {s}
                    </option>
                  ))}
                </select>
              </label>
              <label className="block text-xs uppercase tracking-[0.12em] text-text-muted">
                Description
                <input
                  className="mt-1 w-full bg-surface border border-surface text-text-main px-3 py-2 rounded"
                  value={ipForm.description}
                  onChange={(e) =>
                    setIpForm((f) => ({ ...f, description: e.target.value }))
                  }
                />
              </label>
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button onClick={closeIP} className="terminal-button py-1 px-4 text-xs">
                Cancel
              </button>
              <button
                onClick={submitIP}
                className="border border-signal-green text-signal-green py-1 px-4 text-xs uppercase tracking-[0.14em] hover:bg-signal-green/10"
              >
                Save
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
