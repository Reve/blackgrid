import React from 'react';
import { BrowserRouter, Routes, Route, Link, useLocation } from 'react-router-dom';
import Dashboard from './pages/Dashboard';
import IPAM from './pages/IPAM';
import Devices from './pages/Devices';
import Monitors from './pages/Monitors';
import Incidents from './pages/Incidents';
import Discovery from './pages/Discovery';
import Settings from './pages/Settings';
import StatusPages from './pages/StatusPages';
import PublicStatusPage from './pages/PublicStatusPage';

function Layout({ children }: { children: React.ReactNode }) {
  const location = useLocation();

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
          <div className="text-xs text-signal-green animate-pulse">SYSTEM_ONLINE</div>
        </div>
      </header>

      <div className="container mx-auto flex-grow flex p-4 gap-6">
        <aside className="w-48 flex-shrink-0">
          <nav className="space-y-2">
            {navItems.map((item) => (
              <Link
                key={item.path}
                to={item.path}
                className={`block px-4 py-2 rounded text-sm transition-colors ${
                  location.pathname === item.path
                    ? 'bg-signal-green/10 text-signal-green border-l-2 border-signal-green'
                    : 'text-text-muted hover:text-text-main hover:bg-surface'
                }`}
              >
                {item.label}
              </Link>
            ))}
          </nav>
        </aside>

        <main className="flex-grow">
          {children}
        </main>
      </div>
    </div>
  );
}

function AdminApp() {
  return (
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
  );
}

function App() {
  return (
    <BrowserRouter>
      <Routes>
        {/* Public status pages render without the admin chrome. */}
        <Route path="/status/:slug" element={<PublicStatusPage />} />
        <Route path="*" element={<AdminApp />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
