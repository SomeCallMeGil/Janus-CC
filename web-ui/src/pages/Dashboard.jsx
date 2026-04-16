import { useEffect, useState } from 'react';
import { api } from '../lib/api';

export default function Dashboard() {
  const [health, setHealth] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    api.get('/health')
      .then(setHealth)
      .catch((err) => setError(err.message));
  }, []);

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-gray-800">Overview</h2>
        <p className="text-sm text-gray-500 mt-1">Janus ransomware simulation platform</p>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <StatCard label="API Status" value={health ? '✅ Online' : error ? '❌ Offline' : '…'} />
        <StatCard label="Version" value="v3.2.0" />
        <StatCard label="Mode" value="Standalone" />
      </div>

      {error && (
        <p className="text-sm text-red-500">Backend unreachable: {error}</p>
      )}
    </div>
  );
}

function StatCard({ label, value }) {
  return (
    <div className="bg-white rounded-lg border border-gray-200 px-4 py-5">
      <p className="text-xs text-gray-500 uppercase tracking-wide">{label}</p>
      <p className="mt-1 text-lg font-semibold text-gray-800">{value}</p>
    </div>
  );
}
