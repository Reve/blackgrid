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
    <div className="min-h-screen bg-background flex flex-col items-center justify-center p-4">
      <div className="w-full max-w-md">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="text-brand text-4xl font-bold tracking-[0.2em] mb-2 font-display">BLACKGRID</div>
          <div className="hud-separator"><span>Homelab Control Plane</span></div>
        </div>

        <div className="panel border-l-2 border-l-accent-orange">
          <h2 className="hud-title text-lg mb-6">■ Authenticate</h2>

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
                className="terminal-input"
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
                className="terminal-input"
              />
            </div>

            <button
              id="login-submit"
              type="submit"
              disabled={loading}
              className="terminal-button-primary mt-2"
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
