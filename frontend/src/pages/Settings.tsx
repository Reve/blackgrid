import { useEffect, useState } from 'react';
import {
  createNotificationChannel,
  deleteNotificationChannel,
  listNotificationChannels,
  testNotificationChannel,
  updateNotificationChannel,
} from '../api/client';
import type { ChannelType, NotificationChannel } from '../api/client';

interface FormState {
  id?: string;
  name: string;
  channel_type: ChannelType;
  enabled: boolean;
  // webhook
  url: string;
  method: string;
  headersJSON: string;
  // smtp
  host: string;
  port: number;
  username: string;
  password: string;
  passwordTouched: boolean;
  from: string;
  toCSV: string;
  use_tls: boolean;
}

const blankForm = (type: ChannelType): FormState => ({
  name: '',
  channel_type: type,
  enabled: true,
  url: '',
  method: 'POST',
  headersJSON: '{}',
  host: '',
  port: 587,
  username: '',
  password: '',
  passwordTouched: false,
  from: '',
  toCSV: '',
  use_tls: true,
});

function channelToForm(c: NotificationChannel): FormState {
  const cfg = c.config ?? {};
  return {
    id: c.id,
    name: c.name,
    channel_type: c.channel_type,
    enabled: c.enabled,
    url: cfg.url ?? '',
    method: cfg.method ?? 'POST',
    headersJSON: JSON.stringify(cfg.headers ?? {}, null, 2),
    host: cfg.host ?? '',
    port: cfg.port ?? 587,
    username: cfg.username ?? '',
    password: cfg.password ?? '',
    passwordTouched: false,
    from: cfg.from ?? '',
    toCSV: Array.isArray(cfg.to) ? cfg.to.join(', ') : '',
    use_tls: cfg.use_tls ?? true,
  };
}

function formToConfig(f: FormState): Record<string, any> {
  if (f.channel_type === 'webhook') {
    let headers: Record<string, string> = {};
    try {
      headers = JSON.parse(f.headersJSON || '{}');
    } catch {
      throw new Error('Headers must be valid JSON');
    }
    return { url: f.url, method: f.method || 'POST', headers };
  }
  const to = f.toCSV.split(',').map(s => s.trim()).filter(Boolean);
  const cfg: Record<string, any> = {
    host: f.host,
    port: Number(f.port),
    username: f.username,
    from: f.from,
    to,
    use_tls: f.use_tls,
  };
  // Only send password if user actually typed one in this session.
  if (f.passwordTouched) cfg.password = f.password;
  return cfg;
}

export default function Settings() {
  const [channels, setChannels] = useState<NotificationChannel[]>([]);
  const [editing, setEditing] = useState<FormState | null>(null);
  const [testResult, setTestResult] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = async () => {
    try {
      const res = await listNotificationChannels();
      setChannels(res.data ?? []);
    } catch (err: any) {
      setError(err?.message ?? 'failed to load channels');
    }
  };

  useEffect(() => { load(); }, []);

  const onSave = async () => {
    if (!editing) return;
    setError(null);
    try {
      const payload = {
        name: editing.name,
        channel_type: editing.channel_type,
        enabled: editing.enabled,
        config: formToConfig(editing),
      };
      if (editing.id) {
        await updateNotificationChannel(editing.id, payload);
      } else {
        await createNotificationChannel(payload);
      }
      setEditing(null);
      await load();
    } catch (err: any) {
      setError(err?.response?.data?.error ?? err?.message ?? 'save failed');
    }
  };

  const onDelete = async (id: string) => {
    if (!confirm('Delete this channel?')) return;
    await deleteNotificationChannel(id);
    await load();
  };

  const onTest = async (id: string) => {
    setTestResult(null);
    try {
      const res = await testNotificationChannel(id);
      const result = res.data;
      setTestResult(`${id}: ${result.status}${result.error ? ` — ${result.error}` : ''}`);
    } catch (err: any) {
      setTestResult(`${id}: error — ${err?.message ?? 'failed'}`);
    }
  };

  const onToggle = async (c: NotificationChannel) => {
    await updateNotificationChannel(c.id, { enabled: !c.enabled });
    await load();
  };

  return (
    <div className="flex flex-col gap-4 h-full">
      <h2 className="text-xl text-signal-green">Settings</h2>

      <div className="panel">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg text-signal-green">Notification Channels</h3>
          <div className="flex gap-2">
            <button
              className="px-2 py-1 border border-signal-green text-signal-green text-xs"
              onClick={() => setEditing(blankForm('webhook'))}
            >
              + WEBHOOK
            </button>
            <button
              className="px-2 py-1 border border-signal-green text-signal-green text-xs"
              onClick={() => setEditing(blankForm('smtp'))}
            >
              + SMTP
            </button>
          </div>
        </div>

        {error && <div className="text-signal-red mb-2 text-sm">{error}</div>}
        {testResult && <div className="text-signal-amber mb-2 text-sm">{testResult}</div>}

        {channels.length === 0 ? (
          <p className="text-text-muted text-sm">No notification channels configured.</p>
        ) : (
          <table className="w-full text-left border-collapse text-sm">
            <thead>
              <tr className="border-b border-border-color text-text-muted">
                <th className="p-2">Name</th>
                <th className="p-2">Type</th>
                <th className="p-2">Enabled</th>
                <th className="p-2">Target</th>
                <th className="p-2">Actions</th>
              </tr>
            </thead>
            <tbody>
              {channels.map(c => (
                <tr key={c.id} className="border-b border-border-color">
                  <td className="p-2 text-text-main">{c.name}</td>
                  <td className="p-2 text-text-muted uppercase text-xs">{c.channel_type}</td>
                  <td className="p-2">
                    <span className={c.enabled ? 'text-signal-green' : 'text-text-muted'}>
                      {c.enabled ? 'YES' : 'NO'}
                    </span>
                  </td>
                  <td className="p-2 text-text-muted">
                    {c.channel_type === 'webhook' ? c.config?.url : c.config?.host}
                  </td>
                  <td className="p-2">
                    <div className="flex gap-1">
                      <button className="text-xs px-2 border border-border-color text-text-muted hover:text-signal-green" onClick={() => setEditing(channelToForm(c))}>EDIT</button>
                      <button className="text-xs px-2 border border-border-color text-text-muted" onClick={() => onToggle(c)}>{c.enabled ? 'DISABLE' : 'ENABLE'}</button>
                      <button className="text-xs px-2 border border-signal-amber text-signal-amber" onClick={() => onTest(c.id)}>TEST</button>
                      <button className="text-xs px-2 border border-signal-red text-signal-red" onClick={() => onDelete(c.id)}>DEL</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {editing && (
        <ChannelForm
          form={editing}
          onChange={setEditing}
          onCancel={() => setEditing(null)}
          onSave={onSave}
        />
      )}
    </div>
  );
}

function ChannelForm({
  form,
  onChange,
  onCancel,
  onSave,
}: {
  form: FormState;
  onChange: (f: FormState) => void;
  onCancel: () => void;
  onSave: () => void;
}) {
  const set = <K extends keyof FormState>(k: K, v: FormState[K]) => onChange({ ...form, [k]: v });

  return (
    <div className="panel">
      <h3 className="text-lg text-signal-green mb-4">
        {form.id ? 'Edit' : 'Create'} {form.channel_type.toUpperCase()} channel
      </h3>
      <div className="grid grid-cols-2 gap-3 text-sm">
        <Field label="Name">
          <input className="w-full bg-bg-deep border border-border-color p-1 text-text-main" value={form.name} onChange={e => set('name', e.target.value)} />
        </Field>
        <Field label="Enabled">
          <input type="checkbox" checked={form.enabled} onChange={e => set('enabled', e.target.checked)} />
        </Field>

        {form.channel_type === 'webhook' && (
          <>
            <Field label="URL"><input className="w-full bg-bg-deep border border-border-color p-1 text-text-main" value={form.url} onChange={e => set('url', e.target.value)} placeholder="https://..." /></Field>
            <Field label="Method">
              <select className="w-full bg-bg-deep border border-border-color p-1 text-text-main" value={form.method} onChange={e => set('method', e.target.value)}>
                <option>POST</option><option>PUT</option><option>PATCH</option>
              </select>
            </Field>
            <Field label="Headers (JSON)" full>
              <textarea className="w-full bg-bg-deep border border-border-color p-1 text-text-main font-mono" rows={4} value={form.headersJSON} onChange={e => set('headersJSON', e.target.value)} />
            </Field>
          </>
        )}

        {form.channel_type === 'smtp' && (
          <>
            <Field label="Host"><input className="w-full bg-bg-deep border border-border-color p-1 text-text-main" value={form.host} onChange={e => set('host', e.target.value)} /></Field>
            <Field label="Port"><input type="number" className="w-full bg-bg-deep border border-border-color p-1 text-text-main" value={form.port} onChange={e => set('port', Number(e.target.value))} /></Field>
            <Field label="Username"><input className="w-full bg-bg-deep border border-border-color p-1 text-text-main" value={form.username} onChange={e => set('username', e.target.value)} /></Field>
            <Field label="Password">
              <input
                type="password"
                className="w-full bg-bg-deep border border-border-color p-1 text-text-main"
                value={form.password}
                onChange={e => onChange({ ...form, password: e.target.value, passwordTouched: true })}
                placeholder={form.id ? 'leave blank to keep existing' : ''}
              />
            </Field>
            <Field label="From"><input className="w-full bg-bg-deep border border-border-color p-1 text-text-main" value={form.from} onChange={e => set('from', e.target.value)} /></Field>
            <Field label="To (comma-separated)"><input className="w-full bg-bg-deep border border-border-color p-1 text-text-main" value={form.toCSV} onChange={e => set('toCSV', e.target.value)} /></Field>
            <Field label="Use TLS">
              <input type="checkbox" checked={form.use_tls} onChange={e => set('use_tls', e.target.checked)} />
            </Field>
          </>
        )}
      </div>
      <div className="flex gap-2 mt-4">
        <button className="px-3 py-1 border border-signal-green text-signal-green text-xs" onClick={onSave}>SAVE</button>
        <button className="px-3 py-1 border border-border-color text-text-muted text-xs" onClick={onCancel}>CANCEL</button>
      </div>
    </div>
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
