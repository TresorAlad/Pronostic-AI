interface LogoProps {
  size?: 'sm' | 'md' | 'lg';
  showText?: boolean;
  className?: string;
}

const sizes = {
  sm: { icon: 32, title: 'text-base', sub: 'text-[10px]' },
  md: { icon: 40, title: 'text-lg', sub: 'text-xs' },
  lg: { icon: 48, title: 'text-xl', sub: 'text-xs' },
};

export default function Logo({ size = 'md', showText = true, className = '' }: LogoProps) {
  const s = sizes[size];

  return (
    <div className={`flex items-center gap-3 ${className}`}>
      <div className="relative shrink-0" style={{ width: s.icon, height: s.icon }}>
        <svg
          viewBox="0 0 64 64"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
          className="h-full w-full"
          style={{ filter: 'drop-shadow(0 4px 24px rgba(16, 185, 129, 0.35))' }}
          aria-hidden
        >
          <defs>
            <linearGradient id="logo-bg" x1="8" y1="4" x2="56" y2="60" gradientUnits="userSpaceOnUse">
              <stop stopColor="#141b2d" />
              <stop offset="1" stopColor="#0b0f19" />
            </linearGradient>
            <linearGradient id="logo-ring" x1="12" y1="8" x2="52" y2="56" gradientUnits="userSpaceOnUse">
              <stop stopColor="#34d399" />
              <stop offset="1" stopColor="#10b981" />
            </linearGradient>
            <linearGradient id="logo-ball" x1="20" y1="18" x2="44" y2="46" gradientUnits="userSpaceOnUse">
              <stop stopColor="#fbbf24" />
              <stop offset="1" stopColor="#f59e0b" />
            </linearGradient>
          </defs>
          <rect x="4" y="4" width="56" height="56" rx="16" fill="url(#logo-bg)" stroke="url(#logo-ring)" strokeWidth="2" />
          <circle cx="32" cy="32" r="14" fill="url(#logo-ball)" opacity="0.95" />
          <path
            d="M32 20 L38 26 L36 34 L28 34 L26 26 Z M32 44 L26 38 L28 30 L36 30 L38 38 Z M20 32 L26 26 L34 28 L34 36 L26 38 Z M44 32 L38 38 L30 36 L30 28 L38 26 Z"
            fill="#0b0f19"
            opacity="0.35"
          />
          <circle cx="32" cy="32" r="3" fill="#10b981" />
          <circle cx="48" cy="16" r="2.5" fill="#34d399" />
          <circle cx="16" cy="48" r="2" fill="#34d399" opacity="0.8" />
          <path d="M48 16 L40 24 M16 48 L24 40" stroke="#34d399" strokeWidth="1.5" strokeLinecap="round" opacity="0.7" />
        </svg>
      </div>
      {showText && (
        <div className="leading-tight">
          <p className={`font-display font-bold tracking-tight text-slate-900 dark:text-white ${s.title}`}>
            Pronostic<span className="text-brand-dark dark:text-brand-light"> AI</span>
          </p>
          <p className={`font-medium uppercase tracking-[0.18em] text-slate-500 ${s.sub}`}>
            Football Intelligence
          </p>
        </div>
      )}
    </div>
  );
}
