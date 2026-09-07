import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { api, formatProbability, type Match, type Prediction } from '../api';
import MarketProbCell from './MarketProbCell';
import { useLiveWebSocket } from '../hooks/useLiveWebSocket';

import { shortMarketLabel } from '../utils/marketLabels';

const PREVIEW_MARKETS = [
  { key: '1x2', label: shortMarketLabel('1x2') },
  { key: 'over_2_5', label: shortMarketLabel('over_2_5') },
  { key: 'btts', label: shortMarketLabel('btts') },
] as const;

function pickLiveMatch(matches: Match[]): Match | undefined {
  if (matches.length === 0) return undefined;
  return [...matches].sort((a, b) => (b.minute ?? 0) - (a.minute ?? 0))[0];
}

function prob1X2(pred?: Prediction): number | undefined {
  if (!pred?.predictions) return undefined;
  const { home_win, draw, away_win } = pred.predictions;
  const values = [home_win, draw, away_win].filter((v): v is number => typeof v === 'number');
  if (values.length === 0) return undefined;
  return Math.max(...values);
}

function marketProb(pred: Prediction | undefined, key: string): number | undefined {
  if (!pred?.predictions) return undefined;
  if (key === '1x2') return prob1X2(pred);
  const value = pred.predictions[key];
  return typeof value === 'number' ? value : undefined;
}

function buildInsight(pred?: Prediction): string {
  if (pred?.ai_analysis?.trim()) return pred.ai_analysis.trim();
  if (!pred?.predictions) return 'Chargement des estimations…';

  const parts: string[] = [];
  const over25 = pred.predictions.over_2_5;
  if (typeof over25 === 'number' && over25 >= 0.55) {
    parts.push(`plus de 2,5 buts probable (${formatProbability(over25)})`);
  }
  const corners = pred.predictions.predicted_total_corners;
  if (typeof corners === 'number' && corners >= 9) {
    parts.push(`corners attendus autour de ${Math.round(corners)}`);
  }
  const btts = pred.predictions.btts;
  if (typeof btts === 'number' && btts >= 0.55) {
    parts.push('les deux équipes peuvent marquer');
  }

  if (parts.length === 0) {
    return 'Estimations recalculées en direct pendant le match.';
  }
  return parts[0].charAt(0).toUpperCase() + parts[0].slice(1) + (parts.length > 1 ? ` · ${parts[1]}` : '') + '.';
}

function mergeLiveLists(wsMatches: Match[], apiMatches: Match[]): Match[] {
  const byId = new Map<string, Match>();
  for (const m of apiMatches) byId.set(m.id, m);
  for (const m of wsMatches) {
    const existing = byId.get(m.id);
    byId.set(m.id, existing ? { ...existing, ...m } : m);
  }
  return [...byId.values()];
}

export default function LandingLivePreview() {
  const { liveMatches: wsMatches, predictions: wsPredictions, probHistory } = useLiveWebSocket();

  const { data: apiLive } = useQuery({
    queryKey: ['landing-live-matches'],
    queryFn: api.getLiveMatches,
    refetchInterval: 15_000,
  });

  const matches = mergeLiveLists(wsMatches, apiLive ?? []);
  const match = pickLiveMatch(matches);

  const { data: apiPrediction } = useQuery({
    queryKey: ['landing-live-prediction', match?.id],
    queryFn: () => api.getPrediction(match!.id),
    enabled: Boolean(match?.id),
    refetchInterval: 30_000,
    retry: 1,
  });

  const prediction = match ? wsPredictions[match.id] ?? apiPrediction : undefined;
  const minute = match?.minute ?? prediction?.minute ?? 0;

  if (!match) {
    return (
      <div className="landing-preview">
        <div className="landing-preview-glow" />
        <div className="landing-preview-card text-center">
          <p className="text-sm font-semibold text-heading">Aucun match en direct</p>
          <p className="mt-2 text-xs text-slate-500">
            Revenez plus tard ou explorez les matchs du jour.
          </p>
          <Link to="/dashboard" className="btn-secondary mt-4 inline-block text-sm">
            Voir les matchs
          </Link>
        </div>
      </div>
    );
  }

  const kickoff = new Date(match.kickoff_at).toLocaleTimeString('fr-FR', {
    hour: '2-digit',
    minute: '2-digit',
  });

  return (
    <div className="landing-preview">
      <div className="landing-preview-glow" />
      <Link to={`/match/${match.id}`} className="landing-preview-card block transition-shadow hover:shadow-lg">
        <div className="mb-4 flex items-center justify-between">
          <span className="text-xs font-bold uppercase tracking-wider text-brand-dark dark:text-brand-light">
            {match.league_name} · {kickoff}
          </span>
          <span className="landing-preview-live">Live {minute}&apos;</span>
        </div>

        <p className="mb-1 text-center font-display text-lg font-semibold text-heading">
          {match.home_team.name} vs {match.away_team.name}
        </p>
        <p className="mb-4 text-center font-display text-2xl font-bold text-brand-dark dark:text-brand-light">
          {match.home_score ?? 0} - {match.away_score ?? 0}
        </p>

        <div className="mb-4 grid grid-cols-3 gap-2 text-center text-sm">
          {PREVIEW_MARKETS.map(({ key, label }) => {
            const prob = marketProb(prediction, key);
            const marketKey = key === '1x2' ? 'home_win' : key;
            if (prob == null) {
              return (
                <div key={key} className="rounded-lg bg-slate-100 py-2 dark:bg-navy-800">
                  <p className="text-xs text-slate-500">{label}</p>
                  <p className="font-semibold text-brand-dark dark:text-brand-light">…</p>
                </div>
              );
            }
            return (
              <MarketProbCell
                key={key}
                label={label}
                probability={prob}
                delta={prediction?.delta?.[marketKey]}
                direction={prediction?.direction?.[marketKey]}
                history={match ? probHistory[match.id]?.[marketKey] : []}
                compact
              />
            );
          })}
        </div>

        <p className="border-t border-slate-200 pt-3 text-xs leading-relaxed text-slate-500 dark:border-navy-600">
          {prediction?.ai_analysis ? 'Analyse IA : ' : 'À retenir : '}
          {buildInsight(prediction)}
        </p>
      </Link>
    </div>
  );
}
