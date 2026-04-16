import { Link } from 'react-router-dom';

export default function NotFound() {
  return (
    <div className="flex flex-col items-center justify-center h-full text-center gap-4">
      <p className="text-6xl font-bold text-gray-200">404</p>
      <p className="text-lg font-medium text-gray-600">Page not found</p>
      <Link to="/" className="text-sm text-primary hover:underline">
        ← Back to Dashboard
      </Link>
    </div>
  );
}
