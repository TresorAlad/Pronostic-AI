import { useEffect, useRef, useState } from 'react';
import { Line, LineChart, ResponsiveContainer } from 'recharts';
import { formatProbability } from '../api';

interface MarketProbCellProps {
  label: string;
  probability: number;
  delta?: number;
  direction?: 'up' | 'down' | 'flat';
  history?: number[];
  compact?: boolean;
}

export default function MarketProbCell({
  label,
  probability,
  delta,
  direction,
  history = [],
  compact = false,
}: MarketProbCellProps) {
  const [flash, setFlash] = useState<'up' | 'down' | null>(null);
  const prevProb = useRef(probability);

  useEffect(() => {
    if (Math.abs(probability - prevProb.current) >= 0.01) {
      setFlash(probability > prevProb.current ? 'up' : 'down');
      prevProb.current = probability;
      const timer = window.setTimeout(() => setFlash(null), 700);
      return () => window.clearTimeout(timer);
    }
    prevProb.current = probability;
  }, [probability]);

  const chartData = history.map((value, index) => ({ index, value }));
  const moveDir = direction ?? (delta != null ? (delta > 0 ? 'up' : delta < 0 ? 'down' : 'flat') : 'flat');

  return (
    <div
      className={`rounded-xl border px-3 py-2 transition-colors duration-300 ${
        flash === 'up'
          ? 'border-brand/50 bg-brand/10'
          : flash === 'down'
            ? 'border-red-500/40 bg-red-500/10'
            : 'border-slate-200 bg-slate-50 dark:border-navy-600 dark:bg-navy-900/80'
      } ${compact ? 'min-w-[88px]' : 'min-w-[120px]'}`}
    >
      <p className="text-xs text-slate-500 truncate">{label}</p>
      <div className="flex items-baseline gap-1">
        <p className="font-display text-lg font-bold text-heading">{formatProbability(probability)}</p>
        {delta != null && Math.abs(delta) >= 0.01 && (
          <span className={`text-xs font-semibold ${moveDir === 'up' ? 'text-brand' : 'text-red-400'}`}>
            {moveDir === 'up' ? '▲' : '▼'} {Math.round(Math.abs(delta) * 100)} %
          </span>
        )}
      </div>
      {chartData.length >= 2 && !compact && (
        <div className="mt-1 h-8 w-full">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={chartData}>
              <Line
                type="monotone"
                dataKey="value"
                stroke={moveDir === 'down' ? '#f87171' : '#10b981'}
                strokeWidth={1.5}
                dot={false}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}
