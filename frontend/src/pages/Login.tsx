import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { login } from '../api/client';
import { useAuth } from '../context/AuthContext';

export default function LoginPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const { refresh } = useAuth();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await login({ email, password });
      await refresh();
      navigate('/');
    } catch (err: any) {
      const msg = err?.response?.data?.message || err?.response?.data?.error || 'Login failed';
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-bg-deep flex flex-col items-center justify-center p-4">
      <div className="w-full max-w-md">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="text-signal-green text-4xl font-bold tracking-widest mb-1">BLACKGRID</div>
          <div className="text-text-muted text-xs tracking-widest">HOMELAB CONTROL PLANE</div>
        </div>

        <div className="panel border border-surface">
          <h2 className="text-signal-green text-lg font-bold mb-6 tracking-wider">AUTHENTICATE</h2>

          {error && (
            <div className="mb-4 p-3 border border-signal-red bg-signal-red/10 text-signal-red text-sm">
              {error}
            </div>
          )}

          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <div className="flex flex-col gap-1">
              <label className="text-text-muted text-xs uppercase tracking-wider">Email</label>
              <input
                id="login-email"
                type="email"
                autoComplete="email"
                required
                value={email}
                onChange={e => setEmail(e.target.value)}
                className="bg-bg-deep border border-border-color text-text-main p-2 text-sm focus:border-signal-green outline-none transition-colors"
                placeholder="admin@homelab.local"
              />
            </div>

            <div className="flex flex-col gap-1">
              <label className="text-text-muted text-xs uppercase tracking-wider">Password</label>
              <input
                id="login-password"
                type="password"
                autoComplete="current-password"
                required
                value={password}
                onChange={e => setPassword(e.target.value)}
                className="bg-bg-deep border border-border-color text-text-main p-2 text-sm focus:border-signal-green outline-none transition-colors"
              />
            </div>

            <button
              id="login-submit"
              type="submit"
              disabled={loading}
              className="mt-2 border border-signal-green text-signal-green px-4 py-2 text-sm font-bold tracking-wider hover:bg-signal-green/10 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loading ? 'AUTHENTICATING...' : 'LOGIN'}
            </button>
          </form>
        </div>

        <div className="mt-4 text-center">
          <span className="text-text-muted text-xs">v0.1.0 · Phase 6</span>
        </div>
      </div>
    </div>
  );
}
