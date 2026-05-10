import React from 'react';
import type { ApiErrorDetail } from '../api/client';

export const Loading = ({ message = 'Loading...' }: { message?: string }) => (
  <div className="flex-1 flex items-center justify-center p-8 text-text-muted">
    <div className="flex flex-col items-center gap-4">
      <div className="w-8 h-8 border-2 border-brand border-t-transparent rounded-full animate-spin shadow-brand-glow"></div>
      <span className="font-mono text-xs uppercase tracking-[0.25em] text-brand animate-pulse">{message}</span>
    </div>
  </div>
);

export const ErrorState = ({ 
  error, 
  onRetry 
}: { 
  error: string | ApiErrorDetail | null; 
  onRetry?: () => void 
}) => {
  const message = typeof error === 'string' ? error : error?.message || 'An unexpected error occurred';
  const code = typeof error === 'string' ? null : error?.code;
  const requestId = typeof error === 'string' ? null : error?.request_id;

  return (
    <div className="flex-1 flex items-center justify-center p-8">
      <div className="panel border-signal-red max-w-md w-full bg-signal-red/5 border-l-2">
        <h3 className="text-signal-red font-bold mb-2 uppercase tracking-[0.12em]">■ System Error</h3>
        <p className="text-text-main text-sm mb-4">{message}</p>

        {(code || requestId) && (
          <div className="text-[10px] font-mono text-text-muted mb-6 space-y-1 bg-black/40 p-2 border border-line">
            {code && <div>CODE: {code.toUpperCase()}</div>}
            {requestId && <div>REQ_ID: {requestId}</div>}
          </div>
        )}

        {onRetry && (
          <button
            onClick={onRetry}
            className="w-full terminal-button hover:border-signal-red hover:text-signal-red"
          >
            Retry Connection
          </button>
        )}
      </div>
    </div>
  );
};

export const EmptyState = ({ 
  message, 
  icon,
  action
}: { 
  message: string; 
  icon?: React.ReactNode;
  action?: React.ReactNode;
}) => (
  <div className="flex-1 flex flex-col items-center justify-center p-12 text-center">
    {icon && <div className="text-4xl mb-4 opacity-20 text-brand">{icon}</div>}
    <div className="text-accent-orange text-xs mb-3 tracking-[0.3em]">— ■ —</div>
    <p className="text-text-muted text-sm max-w-xs mb-6 italic">"{message}"</p>
    {action}
  </div>
);

export const ConfirmDialog = ({
  isOpen,
  title,
  message,
  onConfirm,
  onCancel,
  confirmLabel = 'Confirm',
  isDestructive = false
}: {
  isOpen: boolean;
  title: string;
  message: string;
  onConfirm: () => void;
  onCancel: () => void;
  confirmLabel?: string;
  isDestructive?: boolean;
}) => {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="panel max-w-sm w-full shadow-2xl border-l-2 border-l-accent-orange">
        <h3 className="hud-title text-lg mb-3">{title}</h3>
        <p className="text-sm text-text-muted mb-6">{message}</p>
        <div className="flex justify-end gap-3">
          <button onClick={onCancel} className="terminal-button py-1 px-4 text-xs">
            Cancel
          </button>
          <button
            onClick={onConfirm}
            className={`py-1 px-4 text-xs uppercase tracking-[0.14em] border rounded-none transition-colors ${
              isDestructive
                ? 'bg-signal-red text-black border-signal-red hover:bg-[#a8003c]'
                : 'bg-brand text-black border-brand hover:bg-brand-dim'
            }`}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
};
