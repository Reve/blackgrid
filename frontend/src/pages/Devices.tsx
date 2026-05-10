import { useEffect, useState } from 'react';
import type { Device, Site, CreateDeviceInput } from '../api/client';
import {
  getDevices,
  createDevice,
  updateDevice,
  deleteDevice,
  getSites,
} from '../api/client';
import { Loading, ErrorState } from '../components/UI';
import { useToast } from '../context/ToastContext';
import { useAuth } from '../context/AuthContext';

const DEVICE_STATUSES = ['active', 'maintenance', 'offline', 'retired'];

interface FormState {
  open: boolean;
  editing?: Device;
  name: string;
  site_id: string;
  description: string;
  status: string;
}

const blankForm: FormState = {
  open: false,
  name: '',
  site_id: '',
  description: '',
  status: 'active',
};

function describeError(e: any): string {
  if (!e) return 'Unknown error';
  if (typeof e === 'string') return e;
  return e.message || 'Request failed';
}

export default function Devices() {
  const { isOperator } = useAuth();
  const [devices, setDevices] = useState<Device[]>([]);
  const [sites, setSites] = useState<Site[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<any>(null);
  const [form, setForm] = useState<FormState>(blankForm);
  const { success, error: toastError } = useToast();

  const loadAll = async () => {
    try {
      const [devRes, sitesRes] = await Promise.all([getDevices(), getSites()]);
      setDevices(devRes.data || []);
      setSites(sitesRes.data || []);
      setError(null);
    } catch (e) {
      setError(e);
    }
  };

  useEffect(() => {
    loadAll().finally(() => setLoading(false));
  }, []);

  const openCreate = () => setForm({ ...blankForm, open: true });
  const openEdit = (d: Device) =>
    setForm({
      open: true,
      editing: d,
      name: d.name,
      site_id: d.site_id || '',
      description: d.description || '',
      status: d.status || 'active',
    });
  const close = () => setForm(blankForm);

  const submit = async () => {
    if (!form.name.trim()) {
      toastError('Device name is required');
      return;
    }
    const data: CreateDeviceInput = {
      name: form.name.trim(),
      site_id: form.site_id || null,
      description: form.description || null,
      status: form.status || 'active',
    };
    try {
      if (form.editing) {
        await updateDevice(form.editing.id, data);
        success('Device updated');
      } else {
        await createDevice(data);
        success('Device created');
      }
      close();
      await loadAll();
    } catch (e) {
      toastError(describeError(e));
    }
  };

  const handleDelete = async (d: Device) => {
    if (!confirm(`Delete device "${d.name}"?`)) return;
    try {
      await deleteDevice(d.id);
      success('Device deleted');
      await loadAll();
    } catch (e) {
      toastError(describeError(e));
    }
  };

  const siteName = (id?: string | null) => sites.find((s) => s.id === id)?.name || '-';

  if (loading) return <Loading message="Loading devices..." />;
  if (error) return <ErrorState error={error} onRetry={loadAll} />;

  return (
    <div className="space-y-6">
      <div className="panel">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl text-signal-green">Devices</h2>
          {isOperator && (
            <button
              onClick={openCreate}
              className="text-signal-green border border-signal-green px-3 py-1 text-xs uppercase tracking-[0.14em] hover:bg-signal-green/10"
            >
              + New Device
            </button>
          )}
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b border-surface">
                <th className="p-2">Name</th>
                <th className="p-2">Site</th>
                <th className="p-2">Status</th>
                <th className="p-2">Description</th>
                <th className="p-2">Actions</th>
              </tr>
            </thead>
            <tbody>
              {devices.length === 0 ? (
                <tr>
                  <td colSpan={5} className="p-2 text-text-muted">
                    No devices found
                  </td>
                </tr>
              ) : (
                devices.map((d) => (
                  <tr key={d.id} className="border-b border-surface">
                    <td className="p-2 text-signal-green">{d.name}</td>
                    <td className="p-2 text-text-muted">{siteName(d.site_id)}</td>
                    <td className="p-2">
                      <span
                        className={`px-2 py-1 rounded text-xs ${
                          d.status === 'active'
                            ? 'bg-signal-green/20 text-signal-green'
                            : d.status === 'offline' || d.status === 'retired'
                            ? 'bg-signal-red/20 text-signal-red'
                            : d.status === 'maintenance'
                            ? 'bg-signal-amber/20 text-signal-amber'
                            : 'bg-surface text-text-muted'
                        }`}
                      >
                        {d.status}
                      </span>
                    </td>
                    <td className="p-2 text-text-muted">{d.description}</td>
                    <td className="p-2 space-x-3">
                      {isOperator ? (
                        <>
                          <button
                            onClick={() => openEdit(d)}
                            className="text-signal-green hover:underline"
                          >
                            edit
                          </button>
                          <button
                            onClick={() => handleDelete(d)}
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

      {form.open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4">
          <div className="panel max-w-md w-full">
            <h3 className="text-signal-green mb-4 uppercase tracking-[0.14em]">
              {form.editing ? 'Edit Device' : 'New Device'}
            </h3>
            <div className="space-y-3">
              <label className="block text-xs uppercase tracking-[0.12em] text-text-muted">
                Name
                <input
                  className="mt-1 w-full bg-surface border border-surface text-text-main px-3 py-2 rounded"
                  value={form.name}
                  onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                />
              </label>
              <label className="block text-xs uppercase tracking-[0.12em] text-text-muted">
                Site (optional)
                <select
                  className="mt-1 w-full bg-surface border border-surface text-text-main px-3 py-2 rounded"
                  value={form.site_id}
                  onChange={(e) => setForm((f) => ({ ...f, site_id: e.target.value }))}
                >
                  <option value="">— none —</option>
                  {sites.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.name}
                    </option>
                  ))}
                </select>
              </label>
              <label className="block text-xs uppercase tracking-[0.12em] text-text-muted">
                Status
                <select
                  className="mt-1 w-full bg-surface border border-surface text-text-main px-3 py-2 rounded"
                  value={form.status}
                  onChange={(e) => setForm((f) => ({ ...f, status: e.target.value }))}
                >
                  {DEVICE_STATUSES.map((s) => (
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
                  value={form.description}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, description: e.target.value }))
                  }
                />
              </label>
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button onClick={close} className="terminal-button py-1 px-4 text-xs">
                Cancel
              </button>
              <button
                onClick={submit}
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
