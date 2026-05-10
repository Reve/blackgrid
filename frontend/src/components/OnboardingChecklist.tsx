import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  getSites,
  getPrefixes,
  getDevices,
  getMonitors,
  listNotificationChannels,
  listStatusPages,
} from '../api/client';

const DISMISS_KEY = 'blackgrid.onboarding.dismissed';

interface Step {
  key: string;
  label: string;
  to: string;
  done: boolean;
}

export default function OnboardingChecklist() {
  const [dismissed, setDismissed] = useState<boolean>(
    () => localStorage.getItem(DISMISS_KEY) === '1'
  );
  const [steps, setSteps] = useState<Step[] | null>(null);

  useEffect(() => {
    if (dismissed) return;
    let cancelled = false;
    (async () => {
      try {
        const [sites, prefixes, devices, monitors, channels, pages] = await Promise.all([
          getSites().catch(() => ({ data: [] })),
          getPrefixes().catch(() => ({ data: [] })),
          getDevices().catch(() => ({ data: [] })),
          getMonitors().catch(() => ({ data: [] })),
          listNotificationChannels().catch(() => ({ data: [] })),
          listStatusPages().catch(() => ({ data: [] })),
        ]);
        if (cancelled) return;
        setSteps([
          { key: 'site',     label: 'Create a site',                to: '/ipam',         done: (sites.data ?? []).length > 0 },
          { key: 'prefix',   label: 'Create a prefix',              to: '/ipam',         done: (prefixes.data ?? []).length > 0 },
          { key: 'device',   label: 'Create a device',              to: '/devices',      done: (devices.data ?? []).length > 0 },
          { key: 'monitor',  label: 'Create your first monitor',    to: '/monitors',     done: (monitors.data ?? []).length > 0 },
          { key: 'channel',  label: 'Configure a notification channel', to: '/settings', done: (channels.data ?? []).length > 0 },
          { key: 'page',     label: 'Create a status page',         to: '/status-pages', done: (pages.data ?? []).length > 0 },
        ]);
      } catch {
        // best-effort; if anything fails, hide the checklist rather than spam errors
        setSteps([]);
      }
    })();
    return () => { cancelled = true; };
  }, [dismissed]);

  const dismiss = () => {
    localStorage.setItem(DISMISS_KEY, '1');
    setDismissed(true);
  };

  if (dismissed || !steps) return null;
  if (steps.length === 0) return null;

  const completed = steps.filter(s => s.done).length;
  const total = steps.length;
  // Auto-hide once everything is done; user can also hit dismiss earlier.
  if (completed === total) return null;

  return (
    <div className="panel border-l-2 border-accent-orange">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm uppercase tracking-[0.18em] text-accent-orange">
          Getting Started · {completed} / {total}
        </h3>
        <button
          onClick={dismiss}
          className="text-xs text-text-muted hover:text-signal-red uppercase tracking-[0.12em]"
          aria-label="Dismiss onboarding checklist"
        >
          Dismiss
        </button>
      </div>
      <ul className="flex flex-col gap-1 text-sm">
        {steps.map(s => (
          <li key={s.key} className="flex items-center gap-2">
            <span
              className={`inline-block w-3 h-3 border ${
                s.done ? 'bg-signal-green border-signal-green' : 'border-text-muted'
              }`}
              aria-hidden="true"
            />
            {s.done ? (
              <span className="text-text-muted line-through">{s.label}</span>
            ) : (
              <Link to={s.to} className="text-text-main hover:text-signal-green">
                {s.label}
              </Link>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}
