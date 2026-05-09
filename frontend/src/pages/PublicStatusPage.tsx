import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { getPublicStatusPage } from '../api/client';
import type { AggregateStatus, PublicStatusPageResponse } from '../api/client';

export default function PublicStatusPage() {
  const { slug } = useParams<{ slug: string }>();
  const [data, setData] = useState<PublicStatusPageResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);

  const load = async () => {
    if (!slug) return;
    try {
      const res = await getPublicStatusPage(slug);
      setData(res.data);
      setNotFound(false);
    } catch (err: any) {
      if (err?.response?.status === 404) {
        setNotFound(true);
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    const t = setInterval(load, 30000);
    return () => clearInterval(t);
  }, [slug]);

  if (loading) {
    return (
      <PublicShell>
        <div className="text-text-muted text-center py-12">Loading status…</div>
      </PublicShell>
    );
  }

  if (notFound || !data) {
    return (
      <PublicShell>
        <div className="text-text-muted text-center py-12">
          <div className="text-2xl text-signal-amber mb-2">STATUS PAGE NOT FOUND</div>
          <div className="text-sm">No public status page exists at this URL.</div>
        </div>
      </PublicShell>
    );
  }

  return (
    <PublicShell>
      <header className="text-center mb-8">
        <h1 className="text-3xl text-signal-green tracking-wider">{data.name}</h1>
        {data.description && (
          <p className="text-text-muted text-sm mt-2">{data.description}</p>
        )}
      </header>

      <AggregateBanner status={data.aggregate_status} />

      <section className="mt-8">
        <h2 className="text-sm uppercase text-text-muted mb-3">Services</h2>
        <div className="border border-border-color rounded">
          {data.monitors.length === 0 ? (
            <div className="p-4 text-text-muted text-sm">No services configured.</div>
          ) : (
            <table className="w-full text-left border-collapse text-sm">
              <thead>
                <tr className="border-b border-border-color text-text-muted bg-panel">
                  <th className="p-3">Service</th>
                  <th className="p-3">Type</th>
                  <th className="p-3">Status</th>
                  <th className="p-3">Uptime 24h</th>
                  <th className="p-3">Uptime 30d</th>
                  <th className="p-3">Last checked</th>
                </tr>
              </thead>
              <tbody>
                {data.monitors.map((m, i) => (
                  <tr key={i} className="border-b border-border-color">
                    <td className="p-3 text-text-main">{m.display_name}</td>
                    <td className="p-3 text-text-muted uppercase text-xs">{m.monitor_type}</td>
                    <td className="p-3"><StatusBadge status={m.status} /></td>
                    <td className="p-3 text-text-muted">{formatUptime(m.uptime_24h)}</td>
                    <td className="p-3 text-text-muted">{formatUptime(m.uptime_30d)}</td>
                    <td className="p-3 text-text-muted">
                      {m.last_checked_at ? new Date(m.last_checked_at).toLocaleString() : '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </section>

      {data.incidents && data.incidents.length > 0 && (
        <section className="mt-8">
          <h2 className="text-sm uppercase text-text-muted mb-3">Recent Incidents</h2>
          <div className="border border-border-color rounded">
            <table className="w-full text-left border-collapse text-sm">
              <thead>
                <tr className="border-b border-border-color text-text-muted bg-panel">
                  <th className="p-3">Severity</th>
                  <th className="p-3">Service</th>
                  <th className="p-3">Status</th>
                  <th className="p-3">Started</th>
                  <th className="p-3">Resolved</th>
                  <th className="p-3">Summary</th>
                </tr>
              </thead>
              <tbody>
                {data.incidents.map((i, idx) => {
                  const sevColor =
                    i.severity === 'critical' ? 'text-signal-red' :
                    i.severity === 'warning' ? 'text-signal-amber' :
                    'text-text-muted';
                  return (
                    <tr key={idx} className="border-b border-border-color">
                      <td className={`p-3 uppercase text-xs ${sevColor}`}>{i.severity}</td>
                      <td className="p-3 text-text-main">{i.monitor_display_name}</td>
                      <td className="p-3 text-text-muted uppercase text-xs">{i.status}</td>
                      <td className="p-3 text-text-muted">{i.started_at ? new Date(i.started_at).toLocaleString() : '—'}</td>
                      <td className="p-3 text-text-muted">{i.resolved_at ? new Date(i.resolved_at).toLocaleString() : '—'}</td>
                      <td className="p-3 text-text-muted">{i.summary}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </section>
      )}
    </PublicShell>
  );
}

function PublicShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-bg-deep text-text-main">
      <div className="max-w-4xl mx-auto p-6">{children}</div>
      <footer className="text-center text-text-muted text-xs py-6 border-t border-border-color mt-12">
        Powered by BLACKGRID
      </footer>
    </div>
  );
}

function AggregateBanner({ status }: { status: AggregateStatus }) {
  const config: Record<AggregateStatus, { label: string; color: string; border: string }> = {
    up: { label: 'ALL SYSTEMS OPERATIONAL', color: 'text-signal-green', border: 'border-signal-green' },
    degraded: { label: 'DEGRADED PERFORMANCE', color: 'text-signal-amber', border: 'border-signal-amber' },
    down: { label: 'PARTIAL OUTAGE', color: 'text-signal-red', border: 'border-signal-red' },
    empty: { label: 'NO SERVICES CONFIGURED', color: 'text-text-muted', border: 'border-border-color' },
  };
  const c = config[status];
  return (
    <div className={`border-2 ${c.border} bg-panel py-8 text-center`}>
      <div className={`text-2xl font-bold tracking-widest ${c.color}`}>{c.label}</div>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const color =
    status === 'up' ? 'text-signal-green' :
    status === 'down' ? 'text-signal-red' :
    status === 'degraded' || status === 'unknown' ? 'text-signal-amber' :
    'text-text-muted';
  return <span className={`uppercase text-xs ${color}`}>{status}</span>;
}

function formatUptime(v: number | null): string {
  if (v === null || v === undefined) return '—';
  return v.toFixed(2) + '%';
}
