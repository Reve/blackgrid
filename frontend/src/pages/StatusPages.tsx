import { useEffect, useState } from 'react';
import {
  attachStatusPageMonitor,
  createStatusPage,
  deleteStatusPage,
  getMonitors,
  getStatusPage,
  listStatusPages,
  removeAttachedStatusPageMonitor,
  reorderStatusPageMonitors,
  updateAttachedStatusPageMonitor,
  updateStatusPage,
} from '../api/client';
import type {
  AdminStatusPage,
  AttachedMonitor,
  CreateStatusPageInput,
  Monitor,
  StatusPage,
} from '../api/client';

export default function StatusPages() {
  const [pages, setPages] = useState<StatusPage[]>([]);
  const [selected, setSelected] = useState<AdminStatusPage | null>(null);
  const [creating, setCreating] = useState(false);
  const [createForm, setCreateForm] = useState<CreateStatusPageInput>({ name: '', slug: '' });
  const [error, setError] = useState<string | null>(null);
  const [allMonitors, setAllMonitors] = useState<Monitor[]>([]);

  const loadList = async () => {
    try {
      const res = await listStatusPages();
      setPages(res.data ?? []);
    } catch (err: any) {
      setError(err?.response?.data?.error ?? 'failed to load status pages');
    }
  };

  const loadMonitors = async () => {
    try {
      const res = await getMonitors();
      setAllMonitors(res.data ?? []);
    } catch {
      // non-fatal
    }
  };

  useEffect(() => {
    loadList();
    loadMonitors();
  }, []);

  const onCreate = async () => {
    setError(null);
    try {
      await createStatusPage(createForm);
      setCreating(false);
      setCreateForm({ name: '', slug: '' });
      await loadList();
    } catch (err: any) {
      setError(err?.response?.data?.error ?? 'create failed');
    }
  };

  const onSelect = async (id: string) => {
    setError(null);
    try {
      const res = await getStatusPage(id);
      setSelected(res.data);
    } catch (err: any) {
      setError(err?.response?.data?.error ?? 'load page failed');
    }
  };

  const onDelete = async (id: string) => {
    if (!confirm('Delete this status page?')) return;
    await deleteStatusPage(id);
    if (selected?.page.id === id) setSelected(null);
    await loadList();
  };

  const onTogglePublic = async (page: StatusPage) => {
    await updateStatusPage(page.id, { public: !page.public });
    await loadList();
    if (selected?.page.id === page.id) await onSelect(page.id);
  };

  const onToggleUptime = async (page: StatusPage) => {
    await updateStatusPage(page.id, { show_uptime: !page.show_uptime });
    await loadList();
    if (selected?.page.id === page.id) await onSelect(page.id);
  };

  const onToggleIncidents = async (page: StatusPage) => {
    await updateStatusPage(page.id, { show_incidents: !page.show_incidents });
    await loadList();
    if (selected?.page.id === page.id) await onSelect(page.id);
  };

  const reloadSelected = async () => {
    if (selected) await onSelect(selected.page.id);
  };

  return (
    <div className="flex flex-col gap-4 h-full">
      <div className="flex items-center justify-between">
        <h2 className="text-xl text-signal-green">Status Pages</h2>
        <button
          className="px-2 py-1 border border-signal-green text-signal-green text-xs"
          onClick={() => setCreating(v => !v)}
        >
          + NEW STATUS PAGE
        </button>
      </div>

      {error && <div className="text-signal-red text-sm">{error}</div>}

      {creating && (
        <div className="panel">
          <h3 className="text-lg text-signal-green mb-3">Create Status Page</h3>
          <div className="grid grid-cols-2 gap-3 text-sm">
            <Field label="Name">
              <input className="w-full bg-bg-deep border border-border-color p-1 text-text-main"
                value={createForm.name} onChange={e => setCreateForm({ ...createForm, name: e.target.value })} />
            </Field>
            <Field label="Slug (optional)">
              <input className="w-full bg-bg-deep border border-border-color p-1 text-text-main"
                value={createForm.slug ?? ''} onChange={e => setCreateForm({ ...createForm, slug: e.target.value })}
                placeholder="auto-generated from name" />
            </Field>
            <Field label="Description" full>
              <textarea className="w-full bg-bg-deep border border-border-color p-1 text-text-main"
                value={createForm.description ?? ''} onChange={e => setCreateForm({ ...createForm, description: e.target.value })} />
            </Field>
            <Field label="Public">
              <input type="checkbox" checked={createForm.public ?? false}
                onChange={e => setCreateForm({ ...createForm, public: e.target.checked })} />
            </Field>
            <Field label="Show uptime">
              <input type="checkbox" checked={createForm.show_uptime ?? true}
                onChange={e => setCreateForm({ ...createForm, show_uptime: e.target.checked })} />
            </Field>
            <Field label="Show incidents">
              <input type="checkbox" checked={createForm.show_incidents ?? true}
                onChange={e => setCreateForm({ ...createForm, show_incidents: e.target.checked })} />
            </Field>
          </div>
          <div className="flex gap-2 mt-3">
            <button className="px-3 py-1 border border-signal-green text-signal-green text-xs" onClick={onCreate}>SAVE</button>
            <button className="px-3 py-1 border border-border-color text-text-muted text-xs" onClick={() => setCreating(false)}>CANCEL</button>
          </div>
        </div>
      )}

      <div className="panel">
        <h3 className="text-lg text-signal-green mb-3">Pages ({pages.length})</h3>
        {pages.length === 0 ? (
          <p className="text-text-muted text-sm">No status pages configured.</p>
        ) : (
          <table className="w-full text-left border-collapse text-sm">
            <thead>
              <tr className="border-b border-border-color text-text-muted">
                <th className="p-2">Name</th>
                <th className="p-2">Slug</th>
                <th className="p-2">Visibility</th>
                <th className="p-2">Uptime</th>
                <th className="p-2">Incidents</th>
                <th className="p-2">Actions</th>
              </tr>
            </thead>
            <tbody>
              {pages.map(p => (
                <tr key={p.id} className="border-b border-border-color">
                  <td className="p-2 text-text-main">{p.name}</td>
                  <td className="p-2 text-text-muted font-mono text-xs">{p.slug}</td>
                  <td className="p-2">
                    <button
                      className={`text-xs px-2 border ${p.public ? 'border-signal-green text-signal-green' : 'border-border-color text-text-muted'}`}
                      onClick={() => onTogglePublic(p)}
                    >
                      {p.public ? 'PUBLIC' : 'PRIVATE'}
                    </button>
                  </td>
                  <td className="p-2">
                    <button
                      className={`text-xs px-2 border ${p.show_uptime ? 'border-signal-green text-signal-green' : 'border-border-color text-text-muted'}`}
                      onClick={() => onToggleUptime(p)}
                    >
                      {p.show_uptime ? 'ON' : 'OFF'}
                    </button>
                  </td>
                  <td className="p-2">
                    <button
                      className={`text-xs px-2 border ${p.show_incidents ? 'border-signal-green text-signal-green' : 'border-border-color text-text-muted'}`}
                      onClick={() => onToggleIncidents(p)}
                    >
                      {p.show_incidents ? 'ON' : 'OFF'}
                    </button>
                  </td>
                  <td className="p-2">
                    <div className="flex gap-1">
                      <button className="text-xs px-2 border border-border-color text-text-muted hover:text-signal-green"
                        onClick={() => onSelect(p.id)}>MANAGE</button>
                      {p.public && (
                        <a className="text-xs px-2 border border-border-color text-text-muted hover:text-signal-green"
                          href={`/status/${p.slug}`} target="_blank" rel="noreferrer">PREVIEW</a>
                      )}
                      <button className="text-xs px-2 border border-signal-red text-signal-red"
                        onClick={() => onDelete(p.id)}>DEL</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {selected && (
        <StatusPageDetail
          page={selected}
          allMonitors={allMonitors}
          onClose={() => setSelected(null)}
          onChanged={reloadSelected}
        />
      )}
    </div>
  );
}

function StatusPageDetail({
  page,
  allMonitors,
  onClose,
  onChanged,
}: {
  page: AdminStatusPage;
  allMonitors: Monitor[];
  onClose: () => void;
  onChanged: () => void | Promise<void>;
}) {
  const [attachId, setAttachId] = useState('');
  const [attachName, setAttachName] = useState('');
  const [error, setError] = useState<string | null>(null);

  const attachedIds = new Set(page.monitors.map(m => m.monitor.id));
  const candidates = allMonitors.filter(m => !attachedIds.has(m.id));

  const onAttach = async () => {
    setError(null);
    if (!attachId) return;
    try {
      await attachStatusPageMonitor(page.page.id, {
        monitor_id: attachId,
        display_name: attachName || undefined,
      });
      setAttachId('');
      setAttachName('');
      await onChanged();
    } catch (err: any) {
      setError(err?.response?.data?.error ?? 'attach failed');
    }
  };

  const onRemove = async (monitorId: string) => {
    await removeAttachedStatusPageMonitor(page.page.id, monitorId);
    await onChanged();
  };

  const onUpdateRow = async (am: AttachedMonitor, displayName: string, displayOrder: number) => {
    await updateAttachedStatusPageMonitor(page.page.id, am.monitor.id, {
      display_name: displayName,
      display_order: displayOrder,
    });
    await onChanged();
  };

  const onMove = async (idx: number, dir: -1 | 1) => {
    const target = idx + dir;
    if (target < 0 || target >= page.monitors.length) return;
    const ids = page.monitors.map(m => m.monitor.id);
    [ids[idx], ids[target]] = [ids[target], ids[idx]];
    await reorderStatusPageMonitors(page.page.id, ids);
    await onChanged();
  };

  return (
    <div className="panel">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-lg text-signal-green">
          Manage: {page.page.name} <span className="text-text-muted text-xs">/{page.page.slug}</span>
        </h3>
        <button className="text-xs px-2 border border-border-color text-text-muted" onClick={onClose}>CLOSE</button>
      </div>

      {error && <div className="text-signal-red text-sm mb-2">{error}</div>}

      <div className="mb-4">
        <h4 className="text-sm text-text-muted uppercase mb-2">Attach Monitor</h4>
        <div className="flex gap-2 items-end">
          <div className="flex-1">
            <select
              className="w-full bg-bg-deep border border-border-color p-1 text-text-main text-sm"
              value={attachId}
              onChange={e => setAttachId(e.target.value)}
            >
              <option value="">— select monitor —</option>
              {candidates.map(m => (
                <option key={m.id} value={m.id}>{m.name} ({m.monitor_type})</option>
              ))}
            </select>
          </div>
          <div className="flex-1">
            <input
              className="w-full bg-bg-deep border border-border-color p-1 text-text-main text-sm"
              placeholder="display name (optional)"
              value={attachName}
              onChange={e => setAttachName(e.target.value)}
            />
          </div>
          <button className="px-3 py-1 border border-signal-green text-signal-green text-xs" onClick={onAttach}>+ ATTACH</button>
        </div>
      </div>

      <h4 className="text-sm text-text-muted uppercase mb-2">Attached Monitors ({page.monitors.length})</h4>
      {page.monitors.length === 0 ? (
        <p className="text-text-muted text-sm">No monitors attached.</p>
      ) : (
        <table className="w-full text-left border-collapse text-sm">
          <thead>
            <tr className="border-b border-border-color text-text-muted">
              <th className="p-2 w-16">Order</th>
              <th className="p-2">Monitor</th>
              <th className="p-2">Display name</th>
              <th className="p-2">Status</th>
              <th className="p-2">Actions</th>
            </tr>
          </thead>
          <tbody>
            {page.monitors.map((am, idx) => (
              <AttachedRow
                key={am.monitor.id}
                am={am}
                idx={idx}
                total={page.monitors.length}
                onMove={onMove}
                onRemove={onRemove}
                onUpdate={onUpdateRow}
              />
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

function AttachedRow({
  am, idx, total, onMove, onRemove, onUpdate,
}: {
  am: AttachedMonitor;
  idx: number;
  total: number;
  onMove: (idx: number, dir: -1 | 1) => void;
  onRemove: (monitorId: string) => void;
  onUpdate: (am: AttachedMonitor, displayName: string, displayOrder: number) => void;
}) {
  const [name, setName] = useState(am.display_name ?? '');
  const [order, setOrder] = useState<number>(am.display_order);
  const sevColor =
    am.monitor.status === 'up' ? 'text-signal-green' :
    am.monitor.status === 'down' ? 'text-signal-red' :
    'text-signal-amber';
  return (
    <tr className="border-b border-border-color">
      <td className="p-2">
        <div className="flex items-center gap-1">
          <button className="text-xs px-1 border border-border-color text-text-muted disabled:opacity-30"
            disabled={idx === 0} onClick={() => onMove(idx, -1)}>↑</button>
          <button className="text-xs px-1 border border-border-color text-text-muted disabled:opacity-30"
            disabled={idx === total - 1} onClick={() => onMove(idx, 1)}>↓</button>
        </div>
      </td>
      <td className="p-2 text-text-main">{am.monitor.name}</td>
      <td className="p-2">
        <div className="flex gap-1 items-center">
          <input className="bg-bg-deep border border-border-color p-1 text-text-main text-xs flex-1"
            value={name} onChange={e => setName(e.target.value)} placeholder={am.monitor.name} />
          <input type="number" className="bg-bg-deep border border-border-color p-1 text-text-main text-xs w-16"
            value={order} onChange={e => setOrder(Number(e.target.value))} />
          <button className="text-xs px-2 border border-border-color text-text-muted"
            onClick={() => onUpdate(am, name, order)}>SAVE</button>
        </div>
      </td>
      <td className={`p-2 uppercase text-xs ${sevColor}`}>{am.monitor.status}</td>
      <td className="p-2">
        <button className="text-xs px-2 border border-signal-red text-signal-red"
          onClick={() => onRemove(am.monitor.id)}>REMOVE</button>
      </td>
    </tr>
  );
}

function Field({ label, children, full }: { label: string; children: React.ReactNode; full?: boolean }) {
  return (
    <label className={`flex flex-col gap-1 ${full ? 'col-span-2' : ''}`}>
      <span className="text-text-muted uppercase text-xs">{label}</span>
      {children}
    </label>
  );
}
