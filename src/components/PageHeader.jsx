export default function PageHeader({ title, description, action }) {
  return (
    <div className="mb-8 flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
      <div>
        <h1 className="text-3xl font-semibold text-slate-950">{title}</h1>
        <p className="mt-2 text-sm text-slate-500">{description}</p>
      </div>
      {action}
    </div>
  )
}
