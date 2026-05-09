import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { setupAdmin } from '../api/client';

export default function SetupPage() {
  const [email, setEmail] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    if (password !== confirm) {
      setError('Passwords do not match');
      return;
    }
    if (password.length < 12) {
      setError('Password must be at least 12 characters');
      return;
    }
    setLoading(true);
    try {
      await setupAdmin({ email, display_name: displayName, password });
      navigate('/login');
    } catch (err: any) {
      const msg = err?.response?.data?.message || err?.response?.data?.error || 'Setup failed';
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-bg-deep flex flex-col items-center justify-center p-4">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <div className="text-signal-green text-4xl font-bold tracking-widest mb-1">BLACKGRID</div>
          <div className="text-text-muted text-xs tracking-widest">INITIAL SETUP</div>
        </div>

        <div className="panel border border-surface">
          <h2 className="text-signal-green text-lg font-bold mb-2 tracking-wider">FIRST ADMIN SETUP</h2>
          <p className="text-text-muted text-sm mb-6">
            No users exist yet. Create the initial administrator account to get started.
          </p>

          {error && (
            <div className="mb-4 p-3 border border-signal-red bg-signal-red/10 text-signal-red text-sm">
              {error}
            </div>
          )}

          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <div className="flex flex-col gap-1">
              <label className="text-text-muted text-xs uppercase tracking-wider">Email</label>
              <input
                id="setup-email"
                type="email"
                required
                value={email}
                onChange={e => setEmail(e.target.value)}
                className="bg-bg-deep border border-border-color text-text-main p-2 text-sm focus:border-signal-green outline-none transition-colors"
                placeholder="admin@homelab.local"
              />
            </div>

            <div className="flex flex-col gap-1">
              <label className="text-text-muted text-xs uppercase tracking-wider">Display Name</label>
              <input
                id="setup-display-name"
                type="text"
                required
                value={displayName}
                onChange={e => setDisplayName(e.target.value)}
                className="bg-bg-deep border border-border-color text-text-main p-2 text-sm focus:border-signal-green outline-none transition-colors"
                placeholder="Admin"
              />
            </div>

            <div className="flex flex-col gap-1">
              <label className="text-text-muted text-xs uppercase tracking-wider">
                Password <span className="text-text-muted normal-case">(min 12 characters)</span>
              </label>
              <input
                id="setup-password"
                type="password"
                required
                minLength={12}
                value={password}
                onChange={e => setPassword(e.target.value)}
                className="bg-bg-deep border border-border-color text-text-main p-2 text-sm focus:border-signal-green outline-none transition-colors"
              />
            </div>

            <div className="flex flex-col gap-1">
              <label className="text-text-muted text-xs uppercase tracking-wider">Confirm Password</label>
              <input
                id="setup-confirm-password"
                type="password"
                required
                value={confirm}
                onChange={e => setConfirm(e.target.value)}
                className="bg-bg-deep border border-border-color text-text-main p-2 text-sm focus:border-signal-green outline-none transition-colors"
              />
            </div>

            <button
              id="setup-submit"
              type="submit"
              disabled={loading}
              className="mt-2 border border-signal-green text-signal-green px-4 py-2 text-sm font-bold tracking-wider hover:bg-signal-green/10 transition-colors disabled:opacity-50"
            >
              {loading ? 'CREATING...' : 'CREATE ADMIN ACCOUNT'}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
