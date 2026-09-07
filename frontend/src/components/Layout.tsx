import { Link, useLocation } from 'react-router-dom';
import Logo from './Logo';
import ThemeToggle from './ThemeToggle';
import UserMenu from './UserMenu';

const navItems = [
  { path: '/dashboard', label: 'Matchs' },
  { path: '/live', label: 'Live' },
  { path: '/performance', label: 'Performance' },
  { path: '/coupon', label: 'Coupon IA' },
  { path: '/my-coupons', label: 'Mes coupons' },
];

export default function Layout({ children }: { children: React.ReactNode }) {
  const location = useLocation();
  const isLanding = location.pathname === '/';

  return (
    <div className="min-h-screen flex flex-col">
      <header className="sticky top-0 z-50 border-b border-slate-200 bg-white/80 backdrop-blur-xl dark:border-navy-600/60 dark:bg-navy-900/75">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-3 md:py-4">
          <Link to="/" className="shrink-0 transition-opacity hover:opacity-90">
            <Logo size="md" />
          </Link>
          <div className="flex items-center gap-2">
            <nav className="flex items-center gap-1 overflow-x-auto">
              {navItems.map((item) => {
                const active = location.pathname === item.path;
                return (
                  <Link
                    key={item.path}
                    to={item.path}
                    className={`nav-link whitespace-nowrap ${active ? 'nav-link-active' : ''}`}
                  >
                    {item.label}
                  </Link>
                );
              })}
            </nav>
            <ThemeToggle />
            <UserMenu />
          </div>
        </div>
      </header>

      <main
        className={
          isLanding
            ? 'mx-auto w-full flex-1'
            : 'mx-auto w-full max-w-6xl flex-1 px-4 py-8 md:py-10'
        }
      >
        {children}
      </main>

      <footer className="mt-auto border-t border-slate-200 bg-white/80 py-6 dark:border-navy-600/60 dark:bg-navy-950/80">
        <div className="mx-auto flex max-w-6xl flex-col items-center gap-2 px-4 text-center">
          <Logo size="sm" showText={false} />
          <p className="text-xs text-slate-500">
            Estimations statistiques · Jeu responsable · Aucune garantie de gain
          </p>
        </div>
      </footer>
    </div>
  );
}
