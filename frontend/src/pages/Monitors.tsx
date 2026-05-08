import { useEffect, useState } from 'react';
import type { Monitor, MonitorResult } from "../api/client";
import { getMonitors, createMonitor, updateMonitor, deleteMonitor, pauseMonitor, resumeMonitor, testMonitor, getMonitorResults } from '../api/client';

export default function Monitors() {
  const [monitors, setMonitors] = useState<Monitor[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedMonitor, setSelectedMonitor] = useState<Monitor | null>(null);
  const [results, setResults] = useState<MonitorResult[]>([]);

  const [showForm, setShowForm] = useState(false);
  const [formData, setFormData] = useState<Partial<Monitor>>({
    name: '',
    monitor_type: 'http',
    target: '',
    interval_seconds: 60,
    timeout_seconds: 10,
    retry_count: 3,
  });

  useEffect(() => {
    fetchMonitors();
  }, []);

  const fetchMonitors = async () => {
    try {
      const res = await getMonitors();
      setMonitors(res.data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleTest = async (m: Monitor) => {
    try {
      await testMonitor(m.id);
      fetchMonitors();
      if (selectedMonitor?.id === m.id) {
        fetchResults(m.id);
      }
    } catch (err) {
      console.error('Test failed', err);
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
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      if (formData.id) {
        await updateMonitor(formData.id, formData);
      } else {
        await createMonitor(formData);
      }
      setShowForm(false);
      fetchMonitors();
    } catch (err) {
      console.error('Save failed', err);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this monitor?')) return;
    try {
      await deleteMonitor(id);
      setSelectedMonitor(null);
      fetchMonitors();
    } catch (err) {
      console.error('Delete failed', err);
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
    });
    setSelectedMonitor(null);
    setShowForm(true);
  };

  const openEditForm = (m: Monitor) => {
    setFormData(m);
    setShowForm(true);
  };

  if (loading) return <div className="p-4 text-text-muted">Loading monitors...</div>;

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
                  <td className="p-2 text-text-muted">{m.target}</td>
                  <td className="p-2 text-text-muted">{m.interval_seconds}s</td>
                  <td className="p-2 flex gap-2">
                    <button className="btn text-xs" onClick={(e) => { e.stopPropagation(); handleTest(m); }}>Test</button>
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
        <div className="w-1/3 panel flex flex-col">
          <div className="flex justify-between items-center mb-4">
            <h3 className="text-lg text-signal-green">{formData.id ? 'Edit Monitor' : 'Create Monitor'}</h3>
            <button className="text-text-muted hover:text-text-main" onClick={() => setShowForm(false)}>✕</button>
          </div>
          <form onSubmit={handleSave} className="flex flex-col gap-4">
            <div>
              <label className="block text-sm text-text-muted mb-1">Name</label>
              <input type="text" className="input w-full" required value={formData.name || ''} onChange={e => setFormData({...formData, name: e.target.value})} />
            </div>
            <div>
              <label className="block text-sm text-text-muted mb-1">Type</label>
              <select className="input w-full" required value={formData.monitor_type || 'http'} onChange={e => setFormData({...formData, monitor_type: e.target.value as any})}>
                <option value="http">HTTP</option>
                <option value="tcp">TCP</option>
                <option value="ping">Ping</option>
              </select>
            </div>
            <div>
              <label className="block text-sm text-text-muted mb-1">Target</label>
              <input type="text" className="input w-full" required placeholder="https://example.com" value={formData.target || ''} onChange={e => setFormData({...formData, target: e.target.value})} />
            </div>
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
            <button type="submit" className="btn mt-4">Save Monitor</button>
          </form>
        </div>
      )}

      {selectedMonitor && !showForm && (
        <div className="w-1/3 panel flex flex-col">
          <div className="flex justify-between items-center mb-4">
            <h3 className="text-lg text-signal-green">{selectedMonitor.name}</h3>
            <div className="flex gap-2">
              <button className="text-text-muted hover:text-text-main text-sm" onClick={() => openEditForm(selectedMonitor)}>Edit</button>
              <button className="text-signal-red hover:text-red-400 text-sm" onClick={() => handleDelete(selectedMonitor.id)}>Delete</button>
              <button className="text-text-muted hover:text-text-main" onClick={() => setSelectedMonitor(null)}>✕</button>
            </div>
          </div>

          <div className="mb-6 text-sm">
            <div className="flex justify-between border-b border-border-color py-1">
              <span className="text-text-muted">Status</span>
              <span className={`uppercase ${selectedMonitor.status === 'up' ? 'text-signal-green' : selectedMonitor.status === 'down' ? 'text-signal-red' : 'text-signal-amber'}`}>{selectedMonitor.status}</span>
            </div>
            <div className="flex justify-between border-b border-border-color py-1">
              <span className="text-text-muted">Target</span>
              <span className="text-text-main">{selectedMonitor.target}</span>
            </div>
            <div className="flex justify-between border-b border-border-color py-1">
              <span className="text-text-muted">Last Checked</span>
              <span className="text-text-main">{selectedMonitor.last_checked_at ? new Date(selectedMonitor.last_checked_at).toLocaleString() : 'Never'}</span>
            </div>
          </div>

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
                    <td className={`p-1 uppercase ${r.status === 'up' ? 'text-signal-green' : 'text-signal-red'}`}>{r.status}</td>
                    <td className="p-1 text-text-muted">{r.latency_ms ? `${r.latency_ms}ms` : '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
