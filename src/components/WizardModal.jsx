import { useState } from 'react'

export default function WizardModal({ open, title, steps, onClose }) {
  const [step, setStep] = useState(0)

  if (!open) return null

  const current = steps[step]

  const close = () => {
    setStep(0)
    onClose()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/30 px-4 py-8">
      <div className="w-full max-w-2xl overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-xl">
        <div className="flex items-center justify-between border-b border-slate-200 px-6 py-4">
          <div className="text-lg font-semibold text-slate-950">{title}</div>
          <button onClick={close} className="rounded-lg px-2 py-1 text-slate-400 hover:bg-slate-100 hover:text-slate-700">
            关闭
          </button>
        </div>

        <div className="border-b border-slate-200 px-6 py-5">
          <div className="flex items-center gap-3">
            {steps.map((item, index) => (
              <div key={item.label} className="flex flex-1 items-center gap-3">
                <button
                  onClick={() => setStep(index)}
                  className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-xs font-semibold ${
                    index <= step ? 'bg-sky-600 text-white' : 'bg-slate-100 text-slate-400'
                  }`}
                >
                  {index + 1}
                </button>
                <div className="hidden text-sm font-medium text-slate-700 sm:block">{item.label}</div>
                {index < steps.length - 1 && <div className="h-px flex-1 bg-slate-200" />}
              </div>
            ))}
          </div>
        </div>

        <div className="min-h-[300px] px-6 py-5">{current.content}</div>

        <div className="flex items-center justify-between border-t border-slate-200 px-6 py-4">
          <button
            onClick={() => setStep((currentStep) => Math.max(currentStep - 1, 0))}
            disabled={step === 0}
            className="rounded-xl bg-slate-100 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-200 disabled:cursor-not-allowed disabled:opacity-40"
          >
            上一步
          </button>
          {step < steps.length - 1 ? (
            <button
              onClick={() => setStep((currentStep) => Math.min(currentStep + 1, steps.length - 1))}
              className="rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700"
            >
              下一步
            </button>
          ) : (
            <button onClick={close} className="rounded-xl bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-700">
              完成
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
