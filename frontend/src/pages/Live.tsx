import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { api, formatProbability } from '../api';
import MatchTeams from '../components/MatchTeams';
import { useLiveWebSocket } from '../hooks/useLiveWebSocket';
import { marketLabel } from '../utils/marketLabels';

function DeltaBadge({ delta }: { delta?: number }) {
  if (delta == null || Math.abs(delta) < 0.01) return null;
  const up = delta > 0;
  return (
    <span className={`ml-1 ${up ? 'text-brand' : 'text-red-400'}`}>
      {up ? '▲' : '▼'} {Math.round(Math.abs(delta) * 100)} pts
    </span>
  );
}

export default function Live() {
  const { liveMatches, predictions, connected } = useLiveWebSocket();

  const { data: apiLive } = useQuery({
    queryKey: ['live-matches'],
    queryFn: api.getLiveMatches,
    refetchInterval: 15000,
  });

  const matches = liveMatches.length > 0 ? liveMatches : (apiLive ?? []);

  return (
    <div>
      <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="page-title">Matchs en direct</h1>
          <p className="page-subtitle">Prédictions recalculées toutes les 10 s · modèle live actif</p>
        </div>
        <div className="flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 py-2 dark:border-navy-600 dark:bg-navy-850">
          <span
            className={`h-2.5 w-2.5 rounded-full ${connected ? 'bg-brand animate-pulse shadow-[0_0_8px_#10b981]' : 'bg-slate-600'}`}
          />
          <span className="text-sm text-slate-400">
            {connected ? 'WebSocket connecté' : 'Déconnecté'}
          </span>
        </div>
      </div>

      {matches.length === 0 && (
        <div className="card text-center py-16">
          <p className="text-slate-400">Aucun match en direct actuellement.</p>
        </div>
      )}

      <div className="grid gap-4">
        {matches.map((match) => {
          const pred = predictions[match.id];
          const entries = pred?.predictions
            ? Object.entries(pred.predictions).slice(0, 5)
            : [];
          return (
            <div key={match.id} className="card relative overflow-hidden transition-all duration-300">
              <div className="absolute left-0 top-0 h-full w-1 bg-gradient-to-b from-red-500 to-red-600" />
              <div className="flex items-center justify-between mb-3 pl-2">
                <div className="flex items-center gap-2">
                  <span className="text-xs font-bold uppercase tracking-wider text-red-400 animate-pulse">
                    Live {match.minute ?? pred?.minute ?? 0}&apos;
                  </span>
                  {pred?.is_live || pred?.model_version?.startsWith('live') ? (
                    <span className="text-xs font-semibold rounded-lg border border-brand/40 bg-brand/10 px-2 py-0.5 text-brand-dark dark:text-brand-light">
                      Modèle live
                    </span>
                  ) : null}
                </div>
                <span className="text-xs text-slate-500">{match.league_name}</span>
              </div>
              <MatchTeams
                home={match.home_team}
                away={match.away_team}
                homeScore={match.home_score}
                awayScore={match.away_score}
                minute={match.minute}
                status="live"
                layout="card"
              />
              {entries.length > 0 && (
                <div className="mt-4 pt-4 border-t border-navy-600 flex flex-wrap gap-2 pl-2">
                  {entries.map(([m, p]) => (
                    <span
                      key={m}
                      className="text-xs rounded-lg border border-slate-200 bg-slate-50 px-2.5 py-1 text-slate-700 dark:border-navy-600 dark:bg-navy-900/80 dark:text-slate-300 transition-colors duration-300"
                    >
                      {marketLabel(m)}: {formatProbability(p as number)}
                      <DeltaBadge delta={pred?.delta?.[m]} />
                    </span>
                  ))}
                </div>
              )}
              <Link
                to={`/match/${match.id}`}
                className="text-sm text-brand-dark hover:text-brand-dark dark:text-brand-light dark:hover:text-brand-glow mt-4 inline-block pl-2 font-medium transition-colors"
              >
                Voir le détail →
              </Link>
            </div>
          );
        })}
      </div>
    </div>
  );
}
