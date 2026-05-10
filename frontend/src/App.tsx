import React from 'react';
import { BrowserRouter, Routes, Route, Navigate, Link, useLocation } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import { EventProvider, useEvents } from './context/EventContext';
import { ToastProvider } from './context/ToastContext';
import { useEffect, useState } from 'react';
import { getSetupStatus, getHealth } from './api/client';

import Dashboard from './pages/Dashboard';
import IPAM from './pages/IPAM';
import Devices from './pages/Devices';
import Monitors from './pages/Monitors';
import Incidents from './pages/Incidents';
import Discovery from './pages/Discovery';
import Settings from './pages/Settings';
import StatusPages from './pages/StatusPages';
import PublicStatusPage from './pages/PublicStatusPage';
import LoginPage from './pages/Login';
import SetupPage from './pages/Setup';

// ---- Layout ----

function Layout({ children }: { children: React.ReactNode }) {
  const location = useLocation();
  const { user, signOut } = useAuth();
  const { isConnected } = useEvents();
  const [version, setVersion] = useState<string>('');
  const [commit, setCommit] = useState<string>('');

  useEffect(() => {
    getHealth()
      .then(r => {
        setVersion(r.data.version || '');
        setCommit(r.data.commit || '');
      })
      .catch(() => { /* health not available — leave version blank */ });
  }, []);

  const navItems = [
    { path: '/', label: 'DASHBOARD' },
    { path: '/ipam', label: 'IPAM' },
    { path: '/devices', label: 'DEVICES' },
    { path: '/monitors', label: 'MONITORS' },
    { path: '/incidents', label: 'INCIDENTS' },
    { path: '/discovery', label: 'DISCOVERY' },
    { path: '/status-pages', label: 'STATUS PAGES' },
    { path: '/settings', label: 'SETTINGS' },
  ];

  return (
    <div className="min-h-screen flex flex-col">
      <header className="border-b border-line bg-black/80 backdrop-blur-sm relative">
        <div className="absolute left-0 top-0 h-full w-1 bg-accent-orange" />
        <div className="container mx-auto px-4 py-3 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="text-brand text-2xl font-bold tracking-[0.18em] font-display">BLACKGRID</span>
            <span className="text-accent-orange text-xs">■</span>
            <span
              className="text-text-muted text-[10px] uppercase tracking-[0.2em] border border-line px-2 py-0.5"
              title={commit ? `commit ${commit}` : undefined}
            >
              {version || 'dev'}
            </span>
          </div>
          <div className="flex items-center gap-4">
            <div className={`flex items-center gap-2 text-[10px] uppercase tracking-[0.2em] ${isConnected ? 'text-brand' : 'text-signal-red'}`}>
              <span className={`status-dot ${isConnected ? 'status-dot-up' : 'status-dot-down'} ${isConnected ? 'animate-pulse' : ''}`} />
              {isConnected ? 'SYSTEM_ONLINE' : 'SYSTEM_OFFLINE'}
            </div>
            {user && (
              <div className="flex items-center gap-2 text-xs">
                <span className="text-text-muted">{user.display_name}</span>
                <span className="text-text-dim">·</span>
                <span className={`uppercase tracking-[0.12em] ${user.role === 'admin' ? 'text-accent-orange' : user.role === 'operator' ? 'text-brand' : 'text-text-muted'}`}>
                  {user.role}
                </span>
                <button
                  id="signout-btn"
                  onClick={signOut}
                  className="ml-2 text-text-muted hover:text-signal-red transition-colors uppercase tracking-[0.12em]"
                >
                  SIGN OUT
                </button>
              </div>
            )}
          </div>
        </div>
      </header>

      <div className="container mx-auto flex-grow flex flex-col md:flex-row p-4 gap-6 min-h-0">
        <aside className="w-full md:w-52 flex-shrink-0">
          <div className="hidden md:flex items-center gap-2 mb-3 pl-3 border-l-2 border-accent-orange">
            <span className="text-text-dim text-[10px] uppercase tracking-[0.25em]">Navigation</span>
          </div>
          <nav className="flex md:flex-col gap-1 overflow-x-auto md:overflow-visible pb-2 md:pb-0">
            {navItems.map((item) => {
              const active = location.pathname === item.path;
              return (
                <Link
                  key={item.path}
                  to={item.path}
                  className={`relative block px-4 py-2 text-xs uppercase tracking-[0.14em] whitespace-nowrap transition-colors ${
                    active
                      ? 'bg-brand-glow text-brand border-l-2 border-brand'
                      : 'text-text-muted hover:text-text-main hover:bg-panel border-l-2 border-transparent'
                  }`}
                >
                  {active && <span className="text-accent-orange mr-2">▸</span>}
                  {item.label}
                </Link>
              );
            })}
          </nav>
        </aside>
        
        <main className="flex-grow min-w-0 overflow-hidden flex flex-col">
          {children}
        </main>
      </div>
    </div>
  );
}

// ---- Auth guard ----

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth();
  const location = useLocation();
  const [setupRequired, setSetupRequired] = useState<boolean | null>(null);
  const [checkingSetup, setCheckingSetup] = useState(true);

  useEffect(() => {
    getSetupStatus()
      .then(r => setSetupRequired(r.data.setup_required))
      .catch(() => setSetupRequired(false))
      .finally(() => setCheckingSetup(false));
  }, []);

  if (loading || checkingSetup) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center text-brand text-sm uppercase tracking-[0.3em] animate-pulse">
        <span className="text-accent-orange mr-2">■</span> INITIALIZING...
      </div>
    );
  }
  if (setupRequired) return <Navigate to="/setup" replace />;
  if (!user) return <Navigate to="/login" state={{ from: location }} replace />;

  return <>{children}</>;
}

// ---- App ----

function AdminApp() {
  return (
    <ProtectedRoute>
      <Layout>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/ipam" element={<IPAM />} />
          <Route path="/devices" element={<Devices />} />
          <Route path="/monitors" element={<Monitors />} />
          <Route path="/incidents" element={<Incidents />} />
          <Route path="/discovery" element={<Discovery />} />
          <Route path="/status-pages" element={<StatusPages />} />
          <Route path="/settings" element={<Settings />} />
        </Routes>
      </Layout>
    </ProtectedRoute>
  );
}

function AppRoutes() {
  const { user } = useAuth();

  return (
    <Routes>
      <Route path="/setup" element={<SetupPage />} />
      <Route path="/login" element={user ? <Navigate to="/" replace /> : <LoginPage />} />
      <Route path="/status/:slug" element={<PublicStatusPage />} />
      <Route path="*" element={<AdminApp />} />
    </Routes>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <ToastProvider>
          <EventProvider>
            <AppRoutes />
          </EventProvider>
        </ToastProvider>
      </AuthProvider>
    </BrowserRouter>
  );
}
