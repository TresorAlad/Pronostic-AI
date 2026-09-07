import { useState } from 'react';

interface TeamLogoProps {
  name: string;
  logoUrl?: string;
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

const sizes = {
  sm: 'w-8 h-8',
  md: 'w-12 h-12',
  lg: 'w-16 h-16',
};

export default function TeamLogo({ name, logoUrl, size = 'md', className = '' }: TeamLogoProps) {
  const [failed, setFailed] = useState(false);

  const initials = name
    .split(' ')
    .slice(0, 2)
    .map((w) => w[0])
    .join('')
    .toUpperCase();

  if (logoUrl && !failed) {
    return (
      <img
        src={logoUrl}
        alt={name}
        title={name}
        className={`${sizes[size]} object-contain rounded-full bg-white/10 p-1 ${className}`}
        loading="lazy"
        onError={() => setFailed(true)}
      />
    );
  }

  return (
    <span
      className={`${sizes[size]} inline-flex items-center justify-center rounded-full bg-pitch-700 text-xs font-bold text-gray-300 ${className}`}
      title={name}
    >
      {initials || '?'}
    </span>
  );
}
