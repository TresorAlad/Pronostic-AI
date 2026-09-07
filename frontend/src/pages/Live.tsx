import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { api } from '../api';
import LiveMarketTicker from '../components/LiveMarketTicker';
import MarketProbCell from '../components/MarketProbCell';
import MatchTeams from '../components/MatchTeams';
import { useLiveWebSocket } from '../hooks/useLiveWebSocket';
import { marketLabel } from '../utils/marketLabels';

export default function Live() {
  const { liveMatches, predictions, probHistory, connected } = useLiveWebSocket();

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
          <p className="page-subtitle">Probabilités mises à jour en direct pendant le match</p>
        </div>
        <div className="flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 py-2 dark:border-navy-600 dark:bg-navy-850">
          <span
            className={`h-2.5 w-2.5 rounded-full ${connected ? 'bg-brand animate-pulse shadow-[0_0_8px_#10b981]' : 'bg-slate-600'}`}
          />
          <span className="text-sm text-slate-400">
            {connected ? 'Flux en direct' : 'Hors ligne'}
          </span>
        </div>
      </div>

      <LiveMarketTicker matches={matches} predictions={predictions} />

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
                <div className="mt-4 pt-4 border-t border-navy-600 grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-2 pl-2">
                  {entries.map(([m, p]) => (
                    <MarketProbCell
                      key={m}
                      label={marketLabel(m)}
                      probability={p as number}
                      delta={pred?.delta?.[m]}
                      direction={pred?.direction?.[m]}
                      history={probHistory[match.id]?.[m]}
                    />
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
