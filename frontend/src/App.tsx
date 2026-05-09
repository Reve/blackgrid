import React from 'react';
import { BrowserRouter, Routes, Route, Navigate, Link, useLocation } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import { EventProvider, useEvents } from './context/EventContext';
import { ToastProvider, useToast } from './context/ToastContext';
import { useEffect, useState } from 'react';
import { getSetupStatus } from './api/client';

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
      <header className="border-b border-surface p-4 bg-panel">
        <div className="container mx-auto flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <span className="text-signal-green text-2xl font-bold">BLACKGRID</span>
            <span className="text-text-muted text-xs bg-surface px-2 py-1 rounded">v0.1.0</span>
          </div>
          <div className="flex items-center gap-4">
            <div className={`text-xs ${isConnected ? 'text-signal-green' : 'text-signal-red'} ${isConnected ? 'animate-pulse' : ''}`}>
              {isConnected ? 'SYSTEM_ONLINE' : 'SYSTEM_OFFLINE'}
            </div>
            {user && (
              <div className="flex items-center gap-2 text-xs">
                <span className="text-text-muted">{user.display_name}</span>
                <span className="text-border-color">·</span>
                <span className={`${user.role === 'admin' ? 'text-signal-amber' : user.role === 'operator' ? 'text-signal-green' : 'text-text-muted'} uppercase`}>
                  {user.role}
                </span>
                <button
                  id="signout-btn"
                  onClick={signOut}
                  className="ml-2 text-text-muted hover:text-signal-red transition-colors"
                >
                  SIGN OUT
                </button>
              </div>
            )}
          </div>
        </div>
      </header>

      <div className="container mx-auto flex-grow flex flex-col md:flex-row p-4 gap-6 min-h-0">
        <aside className="w-full md:w-48 flex-shrink-0">
          <nav className="flex md:flex-col gap-2 overflow-x-auto md:overflow-visible pb-2 md:pb-0">
            {navItems.map((item) => (
              <Link
                key={item.path}
                to={item.path}
                className={`block px-4 py-2 rounded text-sm transition-colors whitespace-nowrap ${
                  location.pathname === item.path
                    ? 'bg-signal-green/10 text-signal-green border-l-0 md:border-l-2 border-b-2 md:border-b-0 border-signal-green'
                    : 'text-text-muted hover:text-text-main hover:bg-surface'
                }`}
              >
                {item.label}
              </Link>
            ))}
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
      <div className="min-h-screen bg-bg-deep flex items-center justify-center text-signal-green text-sm tracking-widest">
        INITIALIZING...
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
