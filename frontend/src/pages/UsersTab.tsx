import { useEffect, useState } from 'react';
import {
  listUsers, createUser, updateUser, deleteUser,
  type AuthUser, type CreateUserRequest, type UpdateUserRequest, type UserRole,
} from '../api/client';

const ROLES: UserRole[] = ['admin', 'operator', 'viewer'];

interface UserForm {
  id?: string;
  email: string;
  display_name: string;
  role: UserRole;
  enabled: boolean;
  password: string;
}

const blank = (): UserForm => ({ email: '', display_name: '', role: 'viewer', enabled: true, password: '' });

function fromUser(u: AuthUser): UserForm {
  return { id: u.id, email: u.email, display_name: u.display_name, role: u.role, enabled: u.enabled, password: '' };
}

export default function UsersTab() {
  const [users, setUsers] = useState<AuthUser[]>([]);
  const [editing, setEditing] = useState<UserForm | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const load = async () => {
    try { setUsers((await listUsers()).data ?? []); } catch (e: any) { setError(e?.message); }
  };
  useEffect(() => { load(); }, []);

  const onSave = async () => {
    if (!editing) return;
    setSaving(true); setError(null);
    try {
      if (editing.id) {
        const body: UpdateUserRequest = { display_name: editing.display_name, role: editing.role, enabled: editing.enabled };
        if (editing.password) body.password = editing.password;
        await updateUser(editing.id, body);
      } else {
        const body: CreateUserRequest = { email: editing.email, display_name: editing.display_name, role: editing.role, enabled: editing.enabled, password: editing.password };
        await createUser(body);
      }
      setEditing(null); await load();
    } catch (e: any) { setError(e?.response?.data?.error ?? e?.message ?? 'Save failed'); }
    finally { setSaving(false); }
  };

  const onDelete = async (u: AuthUser) => {
    if (!confirm(`Delete user ${u.email}?`)) return;
    try { await deleteUser(u.id); await load(); } catch (e: any) { setError(e?.response?.data?.error ?? e?.message); }
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg text-signal-green">Users</h3>
        <button className="px-2 py-1 border border-signal-green text-signal-green text-xs" onClick={() => setEditing(blank())}>+ NEW USER</button>
      </div>

      {error && <div className="text-signal-red text-sm">{error}</div>}

      <table className="w-full text-left border-collapse text-sm">
        <thead><tr className="border-b border-border-color text-text-muted">
          <th className="p-2">Email</th><th className="p-2">Name</th><th className="p-2">Role</th><th className="p-2">Enabled</th><th className="p-2">Actions</th>
        </tr></thead>
        <tbody>
          {users.map(u => (
            <tr key={u.id} className="border-b border-border-color">
              <td className="p-2 text-text-main">{u.email}</td>
              <td className="p-2 text-text-muted">{u.display_name}</td>
              <td className="p-2"><span className={`text-xs px-1 border ${u.role === 'admin' ? 'border-signal-amber text-signal-amber' : u.role === 'operator' ? 'border-signal-green text-signal-green' : 'border-border-color text-text-muted'}`}>{u.role.toUpperCase()}</span></td>
              <td className="p-2"><span className={u.enabled ? 'text-signal-green text-xs' : 'text-text-muted text-xs'}>{u.enabled ? 'YES' : 'NO'}</span></td>
              <td className="p-2 flex gap-1">
                <button className="text-xs px-2 border border-border-color text-text-muted hover:text-signal-green" onClick={() => setEditing(fromUser(u))}>EDIT</button>
                <button className="text-xs px-2 border border-signal-red text-signal-red" onClick={() => onDelete(u)}>DEL</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {editing && (
        <div className="panel border border-surface mt-2">
          <h4 className="text-signal-green mb-4">{editing.id ? 'Edit User' : 'Create User'}</h4>
          <div className="grid grid-cols-2 gap-3 text-sm">
            {!editing.id && (
              <label className="flex flex-col gap-1 col-span-2">
                <span className="text-text-muted text-xs uppercase">Email</span>
                <input className="bg-bg-deep border border-border-color p-1 text-text-main" value={editing.email} onChange={e => setEditing({ ...editing, email: e.target.value })} />
              </label>
            )}
            <label className="flex flex-col gap-1">
              <span className="text-text-muted text-xs uppercase">Display Name</span>
              <input className="bg-bg-deep border border-border-color p-1 text-text-main" value={editing.display_name} onChange={e => setEditing({ ...editing, display_name: e.target.value })} />
            </label>
            <label className="flex flex-col gap-1">
              <span className="text-text-muted text-xs uppercase">Role</span>
              <select className="bg-bg-deep border border-border-color p-1 text-text-main" value={editing.role} onChange={e => setEditing({ ...editing, role: e.target.value as UserRole })}>
                {ROLES.map(r => <option key={r} value={r}>{r}</option>)}
              </select>
            </label>
            <label className="flex flex-col gap-1">
              <span className="text-text-muted text-xs uppercase">Password {editing.id ? '(blank = keep)' : ''}</span>
              <input type="password" className="bg-bg-deep border border-border-color p-1 text-text-main" value={editing.password} onChange={e => setEditing({ ...editing, password: e.target.value })} />
            </label>
            <label className="flex items-center gap-2 mt-4">
              <input type="checkbox" checked={editing.enabled} onChange={e => setEditing({ ...editing, enabled: e.target.checked })} />
              <span className="text-text-muted text-xs uppercase">Enabled</span>
            </label>
          </div>
          <div className="flex gap-2 mt-4">
            <button className="px-3 py-1 border border-signal-green text-signal-green text-xs" onClick={onSave} disabled={saving}>{saving ? 'SAVING...' : 'SAVE'}</button>
            <button className="px-3 py-1 border border-border-color text-text-muted text-xs" onClick={() => setEditing(null)}>CANCEL</button>
          </div>
        </div>
      )}
    </div>
  );
}
