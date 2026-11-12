import { createContext, useCallback, useContext, useState } from "react";

const ToastContext = createContext(null);

// Wrap the app with this once. Any component can then call useToast()
// to show a small message in the corner of the screen.
export function ToastProvider({ children }) {
  const [toasts, setToasts] = useState([]);

  const dismiss = useCallback((id) => {
    setToasts((current) => current.filter((toast) => toast.id !== id));
  }, []);

  // type is "info" | "success" | "error" | "danger"
  const notify = useCallback(
    (message, type = "info") => {
      const id = Date.now() + Math.random();
      setToasts((current) => [...current, { id, message, type }]);
      setTimeout(() => dismiss(id), 6000); // disappear after 6 seconds
    },
    [dismiss]
  );

  return (
    <ToastContext.Provider value={notify}>
      {children}
      <div className="toast-stack" aria-live="polite">
        {toasts.map((toast) => (
          <div key={toast.id} className={`toast toast-${toast.type}`}>
            <span>{toast.message}</span>
            <button className="toast-close" onClick={() => dismiss(toast.id)} aria-label="Close">
              ×
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast() {
  return useContext(ToastContext);
}
