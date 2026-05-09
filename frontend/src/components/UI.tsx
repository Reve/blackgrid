import React from 'react';
import type { ApiErrorDetail } from '../api/client';

export const Loading = ({ message = 'Loading...' }: { message?: string }) => (
  <div className="flex-1 flex items-center justify-center p-8 text-text-muted animate-pulse">
    <div className="flex flex-col items-center gap-4">
      <div className="w-8 h-8 border-2 border-signal-green border-t-transparent rounded-full animate-spin"></div>
      <span className="font-mono text-xs uppercase tracking-widest">{message}</span>
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
      <div className="panel border-signal-red max-w-md w-full bg-signal-red/5">
        <h3 className="text-signal-red font-bold mb-2 uppercase tracking-tight">System Error</h3>
        <p className="text-text-main text-sm mb-4">{message}</p>
        
        {(code || requestId) && (
          <div className="text-[10px] font-mono text-text-muted mb-6 space-y-1 bg-black/20 p-2 rounded border border-white/5">
            {code && <div>CODE: {code.toUpperCase()}</div>}
            {requestId && <div>REQ_ID: {requestId}</div>}
          </div>
        )}

        {onRetry && (
          <button 
            onClick={onRetry}
            className="w-full btn btn-outline py-2 text-xs uppercase tracking-widest hover:bg-signal-red hover:text-black transition-colors"
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
    {icon && <div className="text-4xl mb-4 opacity-20">{icon}</div>}
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
      <div className="panel max-w-sm w-full shadow-2xl border-white/10">
        <h3 className="text-lg text-text-main mb-2 font-bold">{title}</h3>
        <p className="text-sm text-text-muted mb-6">{message}</p>
        <div className="flex justify-end gap-3">
          <button onClick={onCancel} className="btn py-1 px-4 text-xs uppercase tracking-widest text-text-muted hover:text-text-main">
            Cancel
          </button>
          <button 
            onClick={onConfirm} 
            className={`btn py-1 px-4 text-xs uppercase tracking-widest ${isDestructive ? 'bg-signal-red text-black hover:bg-red-600' : 'bg-signal-green text-black hover:bg-green-600'}`}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
};
