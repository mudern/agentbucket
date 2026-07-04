import { createContext, useContext, useState, useCallback } from 'react'

const ToastContext = createContext()

let _id = 0

export function ToastProvider({ children }) {
  const [toasts, setToasts] = useState([])

  const toast = useCallback((message, type = 'info') => {
    const id = ++_id
    setToasts((t) => [...t, { id, message, type }])
    setTimeout(() => setToasts((t) => t.filter((x) => x.id !== id)), 3000)
  }, [])

  return (
    <ToastContext.Provider value={toast}>
      {children}
      <div className="fixed bottom-6 right-6 z-[100] flex flex-col gap-2">
        {toasts.map((t) => (
          <div
            key={t.id}
            className={`animate-[slideIn_0.2s_ease-out] rounded-xl px-4 py-3 text-sm font-medium shadow-lg ${
              t.type === 'success' ? 'bg-emerald-600 text-white'
              : t.type === 'error' ? 'bg-red-600 text-white'
              : 'bg-slate-800 text-white dark:bg-slate-100 dark:text-slate-900'
            }`}
          >
            {t.message}
          </div>
        ))}
      </div>
      <style>{`@keyframes slideIn { from { opacity:0; transform:translateY(10px) } to { opacity:1; transform:translateY(0) } }`}</style>
    </ToastContext.Provider>
  )
}

export function useToast() {
  return useContext(ToastContext)
}
