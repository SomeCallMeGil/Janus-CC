import { useLocation } from 'react-router-dom';

const titles = {
  '/': 'Dashboard',
  '/profiles': 'Profiles',
  '/jobs': 'Jobs',
};

export default function Header() {
  const { pathname } = useLocation();
  const title = titles[pathname] ?? 'Janus';

  return (
    <header className="h-14 bg-white border-b border-gray-200 flex items-center px-6 shrink-0">
      <h1 className="text-base font-semibold text-gray-800">{title}</h1>
      <div className="ml-auto flex items-center gap-3">
        <span className="text-xs text-gray-400">API: localhost:8080</span>
        <div className="w-2 h-2 rounded-full bg-green-400" title="Connected" />
      </div>
    </header>
  );
}
