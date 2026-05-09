import { useEffect, useState } from 'react';
import {
  listApiTokens, createApiToken, deleteApiToken, listUsers,
  type ApiToken, type AuthUser, type UserRole,
} from '../api/client';

const ROLES: UserRole[] = ['admin', 'operator', 'viewer'];

export default function ApiTokensTab() {
  const [tokens, setTokens] = useState<ApiToken[]>([]);
  const [users, setUsers] = useState<AuthUser[]>([]);
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState({ user_id: '', name: '', role: 'viewer' as UserRole, expires_at: '' });
  const [newToken, setNewToken] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const load = async () => {
    try {
      const [t, u] = await Promise.all([listApiTokens(), listUsers()]);
      setTokens(t.data ?? []);
      setUsers(u.data ?? []);
    } catch (e: any) { setError(e?.message); }
  };
  useEffect(() => { load(); }, []);

  const onCreate = async () => {
    if (!form.user_id || !form.name) return;
    setSaving(true); setError(null);
    try {
      const res = await createApiToken({
        user_id: form.user_id,
        name: form.name,
        role: form.role,
        expires_at: form.expires_at ? form.expires_at : null,
      });
      setNewToken(res.data.token);
      setCreating(false);
      setForm({ user_id: '', name: '', role: 'viewer', expires_at: '' });
      await load();
    } catch (e: any) { setError(e?.response?.data?.error ?? e?.message ?? 'Create failed'); }
    finally { setSaving(false); }
  };

  const onDelete = async (t: ApiToken) => {
    if (!confirm(`Revoke token "${t.name}"?`)) return;
    try { await deleteApiToken(t.id); await load(); } catch (e: any) { setError(e?.response?.data?.error ?? e?.message); }
  };

  const userEmail = (id: string) => users.find(u => u.id === id)?.email ?? id;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg text-signal-green">API Tokens</h3>
        <button className="px-2 py-1 border border-signal-green text-signal-green text-xs" onClick={() => setCreating(true)}>+ NEW TOKEN</button>
      </div>

      {error && <div className="text-signal-red text-sm">{error}</div>}

      {newToken && (
        <div className="panel border border-signal-amber p-3">
          <p className="text-signal-amber text-xs mb-2 font-bold">⚠ Copy this token now — it will never be shown again:</p>
          <code className="text-text-main text-xs break-all bg-bg-deep p-2 block">{newToken}</code>
          <button className="mt-2 text-xs text-text-muted border border-border-color px-2 py-1" onClick={() => setNewToken(null)}>DISMISS</button>
        </div>
      )}

      <table className="w-full text-left border-collapse text-sm">
        <thead><tr className="border-b border-border-color text-text-muted">
          <th className="p-2">Name</th><th className="p-2">User</th><th className="p-2">Role</th><th className="p-2">Expires</th><th className="p-2">Last Used</th><th className="p-2"></th>
        </tr></thead>
        <tbody>
          {tokens.map(t => (
            <tr key={t.id} className="border-b border-border-color">
              <td className="p-2 text-text-main font-mono">{t.name}</td>
              <td className="p-2 text-text-muted text-xs">{userEmail(t.user_id)}</td>
              <td className="p-2"><span className="text-xs border border-border-color px-1 text-text-muted">{t.role.toUpperCase()}</span></td>
              <td className="p-2 text-text-muted text-xs">{t.expires_at ? new Date(t.expires_at).toLocaleDateString() : '—'}</td>
              <td className="p-2 text-text-muted text-xs">{t.last_used_at ? new Date(t.last_used_at).toLocaleDateString() : 'never'}</td>
              <td className="p-2"><button className="text-xs px-2 border border-signal-red text-signal-red" onClick={() => onDelete(t)}>REVOKE</button></td>
            </tr>
          ))}
          {tokens.length === 0 && <tr><td colSpan={6} className="p-4 text-text-muted text-center text-xs">No tokens</td></tr>}
        </tbody>
      </table>

      {creating && (
        <div className="panel border border-surface mt-2">
          <h4 className="text-signal-green mb-4">Create API Token</h4>
          <div className="grid grid-cols-2 gap-3 text-sm">
            <label className="flex flex-col gap-1">
              <span className="text-text-muted text-xs uppercase">Owner</span>
              <select className="bg-bg-deep border border-border-color p-1 text-text-main" value={form.user_id} onChange={e => setForm({ ...form, user_id: e.target.value })}>
                <option value="">— select user —</option>
                {users.map(u => <option key={u.id} value={u.id}>{u.email}</option>)}
              </select>
            </label>
            <label className="flex flex-col gap-1">
              <span className="text-text-muted text-xs uppercase">Name</span>
              <input className="bg-bg-deep border border-border-color p-1 text-text-main" value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} placeholder="ci-deploy" />
            </label>
            <label className="flex flex-col gap-1">
              <span className="text-text-muted text-xs uppercase">Role</span>
              <select className="bg-bg-deep border border-border-color p-1 text-text-main" value={form.role} onChange={e => setForm({ ...form, role: e.target.value as UserRole })}>
                {ROLES.map(r => <option key={r} value={r}>{r}</option>)}
              </select>
            </label>
            <label className="flex flex-col gap-1">
              <span className="text-text-muted text-xs uppercase">Expires (optional)</span>
              <input type="datetime-local" className="bg-bg-deep border border-border-color p-1 text-text-main" value={form.expires_at} onChange={e => setForm({ ...form, expires_at: e.target.value })} />
            </label>
          </div>
          <div className="flex gap-2 mt-4">
            <button className="px-3 py-1 border border-signal-green text-signal-green text-xs" onClick={onCreate} disabled={saving}>{saving ? 'CREATING...' : 'CREATE'}</button>
            <button className="px-3 py-1 border border-border-color text-text-muted text-xs" onClick={() => setCreating(false)}>CANCEL</button>
          </div>
        </div>
      )}
    </div>
  );
}
