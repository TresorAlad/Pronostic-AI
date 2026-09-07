import type { Match, Prediction } from '../api';
import { formatProbability } from '../api';
import { marketLabel } from '../utils/marketLabels';

interface LiveMarketTickerProps {
  matches: Match[];
  predictions: Record<string, Prediction>;
}

type TickerItem = {
  key: string;
  label: string;
  delta: number;
  minute: number;
};

export default function LiveMarketTicker({ matches, predictions }: LiveMarketTickerProps) {
  const items: TickerItem[] = [];

  for (const match of matches) {
    const pred = predictions[match.id];
    if (!pred?.delta) continue;
    for (const [market, delta] of Object.entries(pred.delta)) {
      if (Math.abs(delta) < 0.02) continue;
      items.push({
        key: `${match.id}-${market}`,
        label: `${match.home_team.name} - ${match.away_team.name} · ${marketLabel(market)} ${formatProbability((pred.predictions?.[market] ?? 0) + delta)}`,
        delta,
        minute: match.minute ?? pred.minute ?? 0,
      });
    }
  }

  items.sort((a, b) => Math.abs(b.delta) - Math.abs(a.delta));
  const top = items.slice(0, 5);
  if (top.length === 0) return null;

  return (
    <div className="mb-6 overflow-hidden rounded-xl border border-slate-200 bg-white dark:border-navy-600 dark:bg-navy-850">
      <div className="flex items-center gap-2 border-b border-slate-200 px-4 py-2 dark:border-navy-600">
        <span className="h-2 w-2 rounded-full bg-brand animate-pulse" />
        <span className="text-xs font-semibold uppercase tracking-wider text-slate-500">Mouvements en direct</span>
      </div>
      <div className="flex gap-4 overflow-x-auto px-4 py-3 text-sm whitespace-nowrap">
        {top.map((item) => (
          <span key={item.key} className="inline-flex items-center gap-1 text-slate-700 dark:text-slate-300">
            <span className={item.delta > 0 ? 'text-brand' : 'text-red-400'}>
              {item.delta > 0 ? '▲' : '▼'} {Math.round(Math.abs(item.delta) * 100)} %
            </span>
            <span>{item.label}</span>
            <span className="text-xs text-slate-500">{item.minute}&apos;</span>
          </span>
        ))}
      </div>
    </div>
  );
}
