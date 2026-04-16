import { useEffect, useState } from 'react';
import { api } from '../lib/api';

export default function Profiles() {
  const [profiles, setProfiles] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    api.get('/profiles')
      .then((data) => setProfiles(data.profiles ?? []))
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-gray-800">Profiles</h2>
        <p className="text-sm text-gray-500 mt-1">Saved generation configurations</p>
      </div>

      {loading && <p className="text-sm text-gray-400">Loading…</p>}
      {error && <p className="text-sm text-red-500">Error: {error}</p>}

      {!loading && !error && profiles.length === 0 && (
        <p className="text-sm text-gray-500">No profiles found.</p>
      )}

      <div className="grid gap-3">
        {profiles.map((p) => (
          <div key={p.id} className="bg-white rounded-lg border border-gray-200 px-4 py-4">
            <div className="flex items-start justify-between">
              <div>
                <p className="font-medium text-gray-800">{p.name}</p>
                {p.description && (
                  <p className="text-sm text-gray-500 mt-0.5">{p.description}</p>
                )}
              </div>
              <span className="text-xs text-gray-400">
                {p.options?.file_count
                  ? `${p.options.file_count.toLocaleString()} files`
                  : p.options?.total_size ?? ''}
              </span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
