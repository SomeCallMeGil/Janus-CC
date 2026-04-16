import { useEffect, useState } from 'react';
import { formatDistanceToNow } from 'date-fns';
import { api } from '../lib/api';

const STATUS_STYLES = {
  running:   'bg-blue-100 text-blue-700',
  completed: 'bg-green-100 text-green-700',
  failed:    'bg-red-100 text-red-700',
  cancelled: 'bg-gray-100 text-gray-600',
};

export default function Jobs() {
  const [jobs, setJobs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    api.get('/scenarios')
      .then((data) => setJobs(data.scenarios ?? []))
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-gray-800">Jobs</h2>
        <p className="text-sm text-gray-500 mt-1">Generation run history</p>
      </div>

      {loading && <p className="text-sm text-gray-400">Loading…</p>}
      {error && <p className="text-sm text-red-500">Error: {error}</p>}

      {!loading && !error && jobs.length === 0 && (
        <p className="text-sm text-gray-500">No jobs yet.</p>
      )}

      <div className="space-y-2">
        {jobs.map((j) => (
          <div key={j.id} className="bg-white rounded-lg border border-gray-200 px-4 py-3 flex items-center gap-4">
            <div className="flex-1 min-w-0">
              <p className="font-medium text-gray-800 truncate">{j.name || j.id}</p>
              {j.created_at && (
                <p className="text-xs text-gray-400">
                  {formatDistanceToNow(new Date(j.created_at), { addSuffix: true })}
                </p>
              )}
            </div>
            <span
              className={`text-xs font-medium px-2 py-0.5 rounded-full ${
                STATUS_STYLES[j.status] ?? 'bg-gray-100 text-gray-600'
              }`}
            >
              {j.status}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
