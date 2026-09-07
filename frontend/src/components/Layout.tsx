import { Link, useLocation } from 'react-router-dom';
import { useLiveWebSocket } from '../hooks/useLiveWebSocket';

const navItems = [
  { path: '/', label: 'Dashboard' },
  { path: '/live', label: 'Live' },
  { path: '/performance', label: 'Performance' },
  { path: '/coupon', label: 'Coupon IA' },
];

export default function Layout({ children }: { children: React.ReactNode }) {
  const location = useLocation();
  const { connected } = useLiveWebSocket();

  return (
    <div className="min-h-screen">
      <header className="border-b border-pitch-700 bg-pitch-800/80 backdrop-blur sticky top-0 z-50">
        <div className="max-w-6xl mx-auto px-4 py-4 flex items-center justify-between">
          <Link to="/" className="flex items-center gap-2">
            <span className="text-2xl">&#9917;</span>
            <span className="font-bold text-lg">Football AI Predictor</span>
          </Link>
          <nav className="flex items-center gap-1">
            {navItems.map((item) => (
              <Link
                key={item.path}
                to={item.path}
                className={`px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                  location.pathname === item.path
                    ? 'bg-accent/20 text-accent'
                    : 'text-gray-400 hover:text-white'
                }`}
              >
                {item.label}
                {item.path === '/live' && connected && (
                  <span className="ml-1 w-2 h-2 bg-red-500 rounded-full inline-block animate-pulse" />
                )}
              </Link>
            ))}
            <Link
              to="/auth"
              className="ml-2 px-3 py-2 text-sm text-gray-400 hover:text-white"
            >
              Compte
            </Link>
          </nav>
        </div>
      </header>
      <main className="max-w-6xl mx-auto px-4 py-8">{children}</main>
      <footer className="border-t border-pitch-700 py-4 text-center text-xs text-gray-500">
        Estimations statistiques - Aucune garantie de gain - Donnees API-Football
      </footer>
    </div>
  );
}
