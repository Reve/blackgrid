import { useEffect, useState } from 'react';
import { getDiagnostics } from '../api/client';
import type { Diagnostics } from '../api/client';

function formatTime(iso: string | null | undefined): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (isNaN(d.getTime())) return '—';
  return d.toLocaleString();
}

function StatusDot({ ok }: { ok: boolean }) {
  return (
    <span
      className={`inline-block w-2 h-2 rounded-full mr-2 ${
        ok ? 'bg-signal-green' : 'bg-signal-red'
      }`}
    />
  );
}

export default function DiagnosticsTab() {
  const [data, setData] = useState<Diagnostics | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await getDiagnostics();
      setData(res.data);
    } catch (err: any) {
      setError(err?.message ?? 'failed to load diagnostics');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    const t = setInterval(load, 10000);
    return () => clearInterval(t);
  }, []);

  if (loading && !data) return <p className="text-text-muted text-sm">Loading diagnostics...</p>;
  if (error) return <p className="text-signal-red text-sm">{error}</p>;
  if (!data) return null;

  const dbOk = data.database.status === 'ok';

  return (
    <div className="flex flex-col gap-4 text-sm">
      <div>
        <h3 className="text-lg text-signal-green mb-2">Application</h3>
        <Row label="Version" value={data.version.version} />
        <Row label="Commit" value={data.version.commit} />
        <Row label="Build Date" value={data.version.build_date} />
        <Row label="Server Time" value={formatTime(data.server_time)} />
        <Row label="Current Role" value={data.current_user_role} />
      </div>

      <div>
        <h3 className="text-lg text-signal-green mb-2">Database</h3>
        <Row
          label="Status"
          value={
            <span>
              <StatusDot ok={dbOk} />
              {data.database.status}
              {data.database.error && (
                <span className="text-signal-red ml-2">{data.database.error}</span>
              )}
            </span>
          }
        />
      </div>

      <div>
        <h3 className="text-lg text-signal-green mb-2">Monitor Scheduler</h3>
        <Row
          label="Running"
          value={<><StatusDot ok={data.monitor_scheduler.running} />{data.monitor_scheduler.running ? 'yes' : 'no'}</>}
        />
        <Row label="Worker Count" value={data.monitor_scheduler.worker_count} />
        <Row label="Last Tick" value={formatTime(data.monitor_scheduler.last_tick_at)} />
        <Row label="Next Due Check" value={formatTime(data.monitor_scheduler.next_due_at)} />
      </div>

      <div>
        <h3 className="text-lg text-signal-green mb-2">Discovery Scheduler</h3>
        <Row
          label="Running"
          value={<><StatusDot ok={data.discovery_scheduler.running} />{data.discovery_scheduler.running ? 'yes' : 'no'}</>}
        />
        <Row label="Worker Count" value={data.discovery_scheduler.worker_count} />
        <Row label="Last Tick" value={formatTime(data.discovery_scheduler.last_tick_at)} />
        <Row label="Running Scans" value={data.discovery_scheduler.running_scans} />
      </div>

      <div>
        <h3 className="text-lg text-signal-green mb-2">Event Stream</h3>
        <Row label="Active SSE Clients" value={data.events.sse_clients} />
      </div>

      <div>
        <h3 className="text-lg text-signal-green mb-2">Retention</h3>
        <Row label="Monitor Results (days)" value={data.retention.monitor_results_days} />
        <Row label="Notification Deliveries (days)" value={data.retention.notification_deliveries_days} />
        <Row label="Audit Log (days)" value={data.retention.audit_log_days} />
        <Row label="Discovery Results (days)" value={data.retention.discovery_results_days} />
        <Row label="Discovery Scans (days)" value={data.retention.discovery_scans_days} />
        <Row label="Cleanup Interval (hours)" value={data.retention.interval_hours} />
      </div>
    </div>
  );
}

function Row({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex justify-between py-1 border-b border-border-color">
      <span className="text-text-muted">{label}</span>
      <span className="text-text-main font-mono">{value}</span>
    </div>
  );
}
