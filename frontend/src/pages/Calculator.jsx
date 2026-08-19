import CalcPanel from '../components/CalcPanel.jsx'

export default function Calculator() {
  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-stone-900">Calculator</h1>
        <p className="text-sm text-stone-600">
          Compute federal and state Social Security taxability. To save results,
          run this from a client's page.
        </p>
      </div>
      <CalcPanel />
    </div>
  )
}
