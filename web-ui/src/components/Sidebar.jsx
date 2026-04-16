import { NavLink } from 'react-router-dom';

const nav = [
  { to: '/', label: 'Dashboard', icon: '📊' },
  { to: '/profiles', label: 'Profiles', icon: '⚙️' },
  { to: '/jobs', label: 'Jobs', icon: '📋' },
];

export default function Sidebar() {
  return (
    <aside className="w-56 shrink-0 bg-gray-900 text-gray-100 flex flex-col min-h-screen">
      <div className="px-4 py-5 border-b border-gray-700">
        <span className="text-lg font-bold tracking-wide text-white">Janus</span>
        <span className="ml-2 text-xs text-gray-400">v3.2</span>
      </div>
      <nav className="flex-1 py-4 space-y-1 px-2">
        {nav.map(({ to, label, icon }) => (
          <NavLink
            key={to}
            to={to}
            end={to === '/'}
            className={({ isActive }) =>
              `flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium transition-colors ${
                isActive
                  ? 'bg-primary text-white'
                  : 'text-gray-300 hover:bg-gray-700 hover:text-white'
              }`
            }
          >
            <span>{icon}</span>
            {label}
          </NavLink>
        ))}
      </nav>
      <div className="px-4 py-3 border-t border-gray-700 text-xs text-gray-500">
        Janus CC — Ransom Simulator
      </div>
    </aside>
  );
}
