import { createContext, useContext, useState, useCallback, type ReactNode } from 'react';

type ToastType = 'info' | 'success' | 'warning' | 'error';

interface Toast {
  id: string;
  message: string;
  type: ToastType;
}

interface ToastContextType {
  toast: (message: string, type?: ToastType) => void;
  info: (message: string) => void;
  success: (message: string) => void;
  warning: (message: string) => void;
  error: (message: string) => void;
}

const ToastContext = createContext<ToastContextType | undefined>(undefined);

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const toast = useCallback((message: string, type: ToastType = 'info') => {
    const id = Math.random().toString(36).substring(2, 9);
    setToasts((prev) => [...prev, { id, message, type }]);
    setTimeout(() => removeToast(id), 5000);
  }, [removeToast]);

  const info = (m: string) => toast(m, 'info');
  const success = (m: string) => toast(m, 'success');
  const warning = (m: string) => toast(m, 'warning');
  const error = (m: string) => toast(m, 'error');

  return (
    <ToastContext.Provider value={{ toast, info, success, warning, error }}>
      {children}
      <div className="fixed bottom-4 right-4 z-[100] flex flex-col gap-2 pointer-events-none">
        {toasts.map((t) => (
          <div
            key={t.id}
            className={`pointer-events-auto min-w-[200px] max-w-sm panel py-3 px-4 shadow-2xl animate-in slide-in-from-right duration-300 flex items-center gap-3 border-l-4 ${
              t.type === 'error' ? 'border-signal-red bg-red-950/20' :
              t.type === 'warning' ? 'border-signal-amber bg-amber-950/20' :
              t.type === 'success' ? 'border-signal-green bg-green-950/20' :
              'border-text-muted bg-surface'
            }`}
          >
            <div className="flex-1 text-sm font-medium">{t.message}</div>
            <button onClick={() => removeToast(t.id)} className="text-text-muted hover:text-text-main">
              &times;
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast() {
  const context = useContext(ToastContext);
  if (!context) throw new Error('useToast must be used within a ToastProvider');
  return context;
}
