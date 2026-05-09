import { useEffect, useState } from 'react';
import type { Monitor, MonitorResult } from "../api/client";
import { getMonitors, createMonitor, updateMonitor, deleteMonitor, pauseMonitor, resumeMonitor, testMonitor, getMonitorResults, rotatePushToken, type ApiErrorDetail } from '../api/client';
import { useEvents } from '../context/EventContext';
import { Loading, ErrorState, ConfirmDialog } from '../components/UI';
import { useToast } from '../context/ToastContext';

export default function Monitors() {
  const [monitors, setMonitors] = useState<Monitor[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedMonitor, setSelectedMonitor] = useState<Monitor | null>(null);
  const [results, setResults] = useState<MonitorResult[]>([]);
  const [error, setError] = useState<ApiErrorDetail | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const { subscribe } = useEvents();
  const { success, error: toastError } = useToast();

  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);
  const [confirmRotate, setConfirmRotate] = useState<string | null>(null);

  const [showForm, setShowForm] = useState(false);
  const [formData, setFormData] = useState<Partial<Monitor>>({
    name: '',
    monitor_type: 'http',
    target: '',
    interval_seconds: 60,
    timeout_seconds: 10,
    retry_count: 3,
    config: {},
  });

  const [generatedToken, setGeneratedToken] = useState<{token: string, url: string} | null>(null);

  useEffect(() => {
    fetchMonitors();
    
    const unsubscribe = subscribe((event) => {
      if (event.type.startsWith('monitor.')) {
        fetchMonitors();
        if (selectedMonitor && event.object_id === selectedMonitor.id) {
          fetchResults(selectedMonitor.id);
        }
      }
    });

    // Handle URL params for pre-filling create form
    const params = new URLSearchParams(window.location.search);
    const type = params.get('type');
    const target = params.get('target');
    const name = params.get('name');
    if (type || target || name) {
        setFormData({
            ...formData,
            monitor_type: (type as any) || 'http',
            target: target || '',
            name: name || '',
        });
        setShowForm(true);
    }

    return () => unsubscribe();
  }, [selectedMonitor, subscribe]);

  const fetchMonitors = async () => {
    try {
      const res = await getMonitors();
      setMonitors(res.data);
      setError(null);
    } catch (err: any) {
      setError(err);
    } finally {
      setLoading(false);
    }
  };

  const handleTest = async (m: Monitor) => {
    if (m.monitor_type === 'push') {
        toastError('Push monitors cannot be tested manually. Use the push endpoint.');
        return;
    }
    try {
      await testMonitor(m.id);
      success(`Test initiated for ${m.name}`);
      fetchMonitors();
      if (selectedMonitor?.id === m.id) {
        fetchResults(m.id);
      }
    } catch (err: any) {
      toastError(err.message || 'Test failed');
    }
  };

  const handleTogglePause = async (m: Monitor) => {
    try {
      if (m.status === 'paused') {
        await resumeMonitor(m.id);
      } else {
        await pauseMonitor(m.id);
      }
      fetchMonitors();
    } catch (err) {
      console.error('Pause/Resume failed', err);
    }
  };

  const fetchResults = async (id: string) => {
    try {
      const res = await getMonitorResults(id);
      setResults(res.data);
    } catch (err) {
      console.error(err);
    }
  };

  const selectMonitor = (m: Monitor) => {
    setSelectedMonitor(m);
    fetchResults(m.id);
    setShowForm(false);
    setGeneratedToken(null);
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      let res;
      if (formData.id) {
        res = await updateMonitor(formData.id, formData);
        success('Monitor updated');
      } else {
        res = await createMonitor(formData);
        success('Monitor created');
        if (formData.monitor_type === 'push' && (res.data as any).generated_push_token) {
            const token = (res.data as any).generated_push_token;
            setGeneratedToken({
                token,
                url: `${window.location.protocol}//${window.location.host}/push/${token}`
            });
        }
      }
      if (!generatedToken) {
          setShowForm(false);
      }
      fetchMonitors();
    } catch (err: any) {
      toastError(err.message || 'Save failed');
    } finally {
      setSubmitting(false);
    }
  };

  const handleRotateToken = async (id: string) => {
      setConfirmRotate(null);
      try {
          const res = await rotatePushToken(id);
          success('Push token rotated');
          setGeneratedToken({
              token: res.data.token,
              url: `${window.location.protocol}//${window.location.host}${res.data.push_url}`
          });
      } catch (err: any) {
          toastError(err.message || 'Rotation failed');
      }
  };

  const handleDelete = async (id: string) => {
    setConfirmDelete(null);
    try {
      await deleteMonitor(id);
      success('Monitor deleted');
      setSelectedMonitor(null);
      fetchMonitors();
    } catch (err: any) {
      toastError(err.message || 'Delete failed');
    }
  };

  const openCreateForm = () => {
    setFormData({
      name: '',
      monitor_type: 'http',
      target: '',
      interval_seconds: 60,
      timeout_seconds: 10,
      retry_count: 3,
      config: {},
    });
    setSelectedMonitor(null);
    setShowForm(true);
    setGeneratedToken(null);
  };

  const openEditForm = (m: Monitor) => {
    setFormData({...m, config: m.config || {}});
    setShowForm(true);
    setGeneratedToken(null);
  };

  if (loading) return <Loading message="Accessing system monitors..." />;
  if (error) return <ErrorState error={error} onRetry={fetchMonitors} />;

  return (
    <div className="flex gap-4 h-full">
      <div className="flex-1 panel flex flex-col">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-xl text-signal-green">Monitors</h2>
          <button className="btn" onClick={openCreateForm}>Add Monitor</button>
        </div>

        <div className="flex-1 overflow-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b border-border-color text-text-muted">
                <th className="p-2">Status</th>
                <th className="p-2">Name</th>
                <th className="p-2">Type</th>
                <th className="p-2">Target</th>
                <th className="p-2">Interval</th>
                <th className="p-2">Actions</th>
              </tr>
            </thead>
            <tbody>
              {monitors.map(m => (
                <tr key={m.id} className="border-b border-border-color hover:bg-bg-light cursor-pointer" onClick={() => selectMonitor(m)}>
                  <td className="p-2">
                    <span className={`px-2 py-1 text-xs uppercase ${m.status === 'up' ? 'text-signal-green' : m.status === 'down' ? 'text-signal-red' : m.status === 'paused' ? 'text-text-muted' : 'text-signal-amber'}`}>
                      {m.status}
                    </span>
                  </td>
                  <td className="p-2 text-text-main">{m.name}</td>
                  <td className="p-2 text-text-muted">{m.monitor_type}</td>
                  <td className="p-2 text-text-muted">{m.monitor_type === 'push' ? 'Passive' : m.target}</td>
                  <td className="p-2 text-text-muted">{m.interval_seconds}s</td>
                  <td className="p-2 flex gap-2">
                    {m.monitor_type !== 'push' && (
                        <button className="btn text-xs" onClick={(e) => { e.stopPropagation(); handleTest(m); }}>Test</button>
                    )}
                    <button className="btn text-xs" onClick={(e) => { e.stopPropagation(); handleTogglePause(m); }}>
                      {m.status === 'paused' ? 'Resume' : 'Pause'}
                    </button>
                  </td>
                </tr>
              ))}
              {monitors.length === 0 && (
                <tr>
                  <td colSpan={6} className="p-4 text-center text-text-muted">No monitors found.</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {showForm && (
        <div className="w-1/3 panel flex flex-col overflow-auto">
          <div className="flex justify-between items-center mb-4">
            <h3 className="text-lg text-signal-green">{formData.id ? 'Edit Monitor' : 'Create Monitor'}</h3>
            <button className="text-text-muted hover:text-text-main" onClick={() => setShowForm(false)}>✕</button>
          </div>

          {generatedToken ? (
              <div className="flex flex-col gap-4">
                  <div className="p-4 bg-bg-light border border-signal-green text-sm">
                      <p className="text-signal-green font-bold mb-2">Push Token Generated!</p>
                      <p className="text-text-muted mb-2">Store this token securely. It will not be shown again.</p>
                      <div className="font-mono text-xs p-2 bg-black break-all mb-2 select-all">{generatedToken.token}</div>
                      <p className="text-text-muted mb-2">Push URL:</p>
                      <div className="font-mono text-xs p-2 bg-black break-all select-all">{generatedToken.url}</div>
                  </div>
                  <button className="btn" onClick={() => setShowForm(false)}>Close</button>
              </div>
          ) : (
            <form onSubmit={handleSave} className="flex flex-col gap-4">
                <div>
                  <label className="block text-sm text-text-muted mb-1">Name</label>
                  <input type="text" className="input w-full" required value={formData.name || ''} onChange={e => setFormData({...formData, name: e.target.value})} />
                </div>
                <div>
                  <label className="block text-sm text-text-muted mb-1">Type</label>
                  <select className="input w-full" required value={formData.monitor_type || 'http'} onChange={e => setFormData({...formData, monitor_type: e.target.value as any, config: {}})}>
                    <option value="http">HTTP</option>
                    <option value="tcp">TCP</option>
                    <option value="ping">Ping</option>
                    <option value="dns">DNS</option>
                    <option value="tls">TLS Certificate</option>
                    <option value="push">Push Heartbeat</option>
                    <option value="postgres">PostgreSQL</option>
                  </select>
                </div>

                {formData.monitor_type !== 'push' && (
                    <div>
                      <label className="block text-sm text-text-muted mb-1">Target</label>
                      <input type="text" className="input w-full" required placeholder={formData.monitor_type === 'postgres' ? 'e.g. pg-server.local' : 'e.g. 10.0.0.1'} value={formData.target || ''} onChange={e => setFormData({...formData, target: e.target.value})} />
                    </div>
                )}

                {/* Type specific config */}
                {formData.monitor_type === 'dns' && (
                    <div className="p-3 bg-bg-light border border-border-color flex flex-col gap-3">
                        <div>
                            <label className="block text-xs text-text-muted mb-1">Record Type</label>
                            <select className="input w-full text-sm" value={formData.config?.record_type || 'A'} onChange={e => setFormData({...formData, config: {...formData.config, record_type: e.target.value}})}>
                                <option value="A">A</option>
                                <option value="AAAA">AAAA</option>
                                <option value="CNAME">CNAME</option>
                                <option value="MX">MX</option>
                                <option value="TXT">TXT</option>
                                <option value="NS">NS</option>
                                <option value="PTR">PTR</option>
                            </select>
                        </div>
                        <div>
                            <label className="block text-xs text-text-muted mb-1">Resolver (Optional)</label>
                            <input type="text" className="input w-full text-sm" placeholder="1.1.1.1:53" value={formData.config?.resolver || ''} onChange={e => setFormData({...formData, config: {...formData.config, resolver: e.target.value}})} />
                        </div>
                        <div>
                            <label className="block text-xs text-text-muted mb-1">Expected Values (Optional, comma separated)</label>
                            <input type="text" className="input w-full text-sm" placeholder="10.0.0.5, 10.0.0.6" value={formData.config?.expected_values?.join(', ') || ''} onChange={e => setFormData({...formData, config: {...formData.config, expected_values: e.target.value.split(',').map(s => s.trim()).filter(s => s)}})} />
                        </div>
                        <div>
                            <label className="block text-xs text-text-muted mb-1">Match Mode</label>
                            <select className="input w-full text-sm" value={formData.config?.match_mode || 'any'} onChange={e => setFormData({...formData, config: {...formData.config, match_mode: e.target.value}})}>
                                <option value="any">Any</option>
                                <option value="all">All</option>
                                <option value="exact">Exact</option>
                            </select>
                        </div>
                    </div>
                )}

                {formData.monitor_type === 'tls' && (
                    <div className="p-3 bg-bg-light border border-border-color flex flex-col gap-3">
                        <div>
                            <label className="block text-xs text-text-muted mb-1">Server Name (SNI)</label>
                            <input type="text" className="input w-full text-sm" placeholder="example.com" value={formData.config?.server_name || ''} onChange={e => setFormData({...formData, config: {...formData.config, server_name: e.target.value}})} />
                        </div>
                        <div className="flex gap-2">
                            <div className="flex-1">
                                <label className="block text-xs text-text-muted mb-1">Warning Days</label>
                                <input type="number" className="input w-full text-sm" value={formData.config?.warning_days ?? 30} onChange={e => setFormData({...formData, config: {...formData.config, warning_days: parseInt(e.target.value)}})} />
                            </div>
                            <div className="flex-1">
                                <label className="block text-xs text-text-muted mb-1">Critical Days</label>
                                <input type="number" className="input w-full text-sm" value={formData.config?.critical_days ?? 7} onChange={e => setFormData({...formData, config: {...formData.config, critical_days: parseInt(e.target.value)}})} />
                            </div>
                        </div>
                        <div className="flex items-center gap-2">
                            <input type="checkbox" checked={formData.config?.verify_tls ?? true} onChange={e => setFormData({...formData, config: {...formData.config, verify_tls: e.target.checked}})} />
                            <label className="text-xs text-text-muted">Verify Certificate Chain</label>
                        </div>
                    </div>
                )}

                {formData.monitor_type === 'push' && (
                    <div className="p-3 bg-bg-light border border-border-color flex flex-col gap-3">
                        <div>
                            <label className="block text-xs text-text-muted mb-1">Grace Period (Seconds)</label>
                            <input type="number" className="input w-full text-sm" required value={formData.config?.grace_seconds ?? 120} onChange={e => setFormData({...formData, config: {...formData.config, grace_seconds: parseInt(e.target.value)}})} />
                        </div>
                        <p className="text-xs text-text-muted italic">Push URL will be shown after saving.</p>
                    </div>
                )}

                {formData.monitor_type === 'postgres' && (
                    <div className="p-3 bg-bg-light border border-border-color flex flex-col gap-3">
                        <div>
                            <label className="block text-xs text-text-muted mb-1">DSN</label>
                            <input type="password" placeholder="postgres://user:pass@host:5432/db" className="input w-full text-sm font-mono" value={formData.config?.dsn || ''} onChange={e => setFormData({...formData, config: {...formData.config, dsn: e.target.value}})} />
                            {formData.id && <p className="text-[10px] text-text-muted mt-1">Leave blank to keep existing DSN.</p>}
                        </div>
                        <div>
                            <label className="block text-xs text-text-muted mb-1">Query</label>
                            <input type="text" className="input w-full text-sm font-mono" value={formData.config?.query || 'SELECT 1'} onChange={e => setFormData({...formData, config: {...formData.config, query: e.target.value}})} />
                        </div>
                    </div>
                )}

                <div className="flex gap-4">
                  <div className="flex-1">
                    <label className="block text-sm text-text-muted mb-1">Interval (s)</label>
                    <input type="number" min="10" className="input w-full" required value={formData.interval_seconds || 60} onChange={e => setFormData({...formData, interval_seconds: parseInt(e.target.value)})} />
                  </div>
                  <div className="flex-1">
                    <label className="block text-sm text-text-muted mb-1">Timeout (s)</label>
                    <input type="number" min="1" className="input w-full" required value={formData.timeout_seconds || 10} onChange={e => setFormData({...formData, timeout_seconds: parseInt(e.target.value)})} />
                  </div>
                  <div className="flex-1">
                    <label className="block text-sm text-text-muted mb-1">Retries</label>
                    <input type="number" min="1" className="input w-full" required value={formData.retry_count || 3} onChange={e => setFormData({...formData, retry_count: parseInt(e.target.value)})} />
                  </div>
                </div>
                <button type="submit" disabled={submitting} className="btn mt-4">
                  {submitting ? 'Saving...' : 'Save Monitor'}
                </button>
            </form>
          )}
        </div>
      )}

      {selectedMonitor && !showForm && (
        <div className="w-1/3 panel flex flex-col overflow-hidden">
          <div className="flex justify-between items-center mb-4">
            <h3 className="text-lg text-signal-green">{selectedMonitor.name}</h3>
            <div className="flex gap-2">
              <button className="text-text-muted hover:text-text-main text-sm" onClick={() => openEditForm(selectedMonitor)}>Edit</button>
              <button className="text-signal-red hover:text-red-400 text-sm" onClick={() => setConfirmDelete(selectedMonitor.id)}>Delete</button>
              <button className="text-text-muted hover:text-text-main" onClick={() => setSelectedMonitor(null)}>✕</button>
            </div>
          </div>

          <div className="mb-6 text-sm">
            <div className="flex justify-between border-b border-border-color py-1">
              <span className="text-text-muted">Status</span>
              <span className={`uppercase font-bold ${selectedMonitor.status === 'up' ? 'text-signal-green' : selectedMonitor.status === 'down' ? 'text-signal-red' : 'text-signal-amber'}`}>{selectedMonitor.status}</span>
            </div>
            <div className="flex justify-between border-b border-border-color py-1">
              <span className="text-text-muted">Type</span>
              <span className="text-text-main uppercase text-xs">{selectedMonitor.monitor_type}</span>
            </div>
            {selectedMonitor.monitor_type !== 'push' && (
                <div className="flex justify-between border-b border-border-color py-1">
                  <span className="text-text-muted">Target</span>
                  <span className="text-text-main">{selectedMonitor.target}</span>
                </div>
            )}
            {selectedMonitor.monitor_type === 'push' && (
                <div className="border-b border-border-color py-2">
                    <div className="flex justify-between mb-1">
                        <span className="text-text-muted">Grace Period</span>
                        <span className="text-text-main">{selectedMonitor.config?.grace_seconds || 120}s</span>
                    </div>
                    <button className="btn text-[10px] w-full" onClick={() => setConfirmRotate(selectedMonitor.id)}>Rotate Push Token</button>
                    {generatedToken && (
                        <div className="mt-2 p-2 bg-black border border-signal-green text-[10px] font-mono break-all select-all">
                            {generatedToken.url}
                        </div>
                    )}
                </div>
            )}
            <div className="flex justify-between border-b border-border-color py-1">
              <span className="text-text-muted">Last Checked</span>
              <span className="text-text-main">{selectedMonitor.last_checked_at ? new Date(selectedMonitor.last_checked_at).toLocaleString() : 'Never'}</span>
            </div>
          </div>

          {/* Type-specific latest details */}
          {results.length > 0 && results[0].details && (
              <div className="mb-6 p-2 bg-bg-light border border-border-color text-xs">
                  <h5 className="text-text-muted uppercase mb-2 text-[10px] font-bold">Latest Check Details</h5>
                  {selectedMonitor.monitor_type === 'dns' && (
                      <>
                        <div className="flex justify-between"><span className="text-text-muted">Records:</span> <span className="text-text-main">{results[0].details.returned_values?.join(', ') || 'None'}</span></div>
                        <div className="flex justify-between"><span className="text-text-muted">Resolver:</span> <span className="text-text-main">{results[0].details.resolver}</span></div>
                      </>
                  )}
                  {selectedMonitor.monitor_type === 'tls' && (
                      <>
                        <div className="flex justify-between"><span className="text-text-muted">Common Name:</span> <span className="text-text-main">{results[0].details.subject}</span></div>
                        <div className="flex justify-between"><span className="text-text-muted">Expires In:</span> <span className={`${results[0].details.days_remaining < 7 ? 'text-signal-red' : 'text-signal-green'}`}>{results[0].details.days_remaining} days</span></div>
                        <div className="flex justify-between"><span className="text-text-muted">Expiry:</span> <span className="text-text-main">{new Date(results[0].details.not_after).toLocaleDateString()}</span></div>
                      </>
                  )}
                  {selectedMonitor.monitor_type === 'postgres' && (
                      <>
                        <div className="flex justify-between"><span className="text-text-muted">Host:</span> <span className="text-text-main">{results[0].details.db_host}</span></div>
                        <div className="flex justify-between"><span className="text-text-muted">Database:</span> <span className="text-text-main">{results[0].details.db_name}</span></div>
                      </>
                  )}
              </div>
          )}

          <h4 className="text-md text-signal-green mb-2">Recent Results</h4>
          <div className="flex-1 overflow-auto border border-border-color">
            <table className="w-full text-left text-sm">
              <thead className="bg-bg-light">
                <tr>
                  <th className="p-1 text-text-muted">Time</th>
                  <th className="p-1 text-text-muted">Status</th>
                  <th className="p-1 text-text-muted">Latency</th>
                </tr>
              </thead>
              <tbody>
                {results.map(r => (
                  <tr key={r.id} className="border-t border-border-color">
                    <td className="p-1 text-text-main">{new Date(r.checked_at).toLocaleTimeString()}</td>
                    <td className={`p-1 uppercase ${r.status === 'up' ? 'text-signal-green' : r.status === 'down' ? 'text-signal-red' : 'text-signal-amber'}`}>{r.status}</td>
                    <td className="p-1 text-text-muted">{r.latency_ms ? `${r.latency_ms}ms` : '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <ConfirmDialog
        isOpen={!!confirmDelete}
        title="Delete Monitor"
        message="Are you sure you want to delete this monitor? This action cannot be undone."
        onConfirm={() => confirmDelete && handleDelete(confirmDelete)}
        onCancel={() => setConfirmDelete(null)}
        confirmLabel="Delete"
        isDestructive
      />

      <ConfirmDialog
        isOpen={!!confirmRotate}
        title="Rotate Push Token"
        message="Are you sure you want to rotate the push token? The old token will stop working immediately."
        onConfirm={() => confirmRotate && handleRotateToken(confirmRotate)}
        onCancel={() => setConfirmRotate(null)}
        confirmLabel="Rotate"
        isDestructive
      />
    </div>
  );
}
