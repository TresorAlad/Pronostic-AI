import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { api, formatProbability } from '../api';
import MatchTeams from '../components/MatchTeams';
import { useLiveWebSocket } from '../hooks/useLiveWebSocket';

export default function Live() {
  const { liveMatches, predictions, connected } = useLiveWebSocket();

  const { data: apiLive } = useQuery({
    queryKey: ['live-matches'],
    queryFn: api.getLiveMatches,
    refetchInterval: 30000,
  });

  const matches = liveMatches.length > 0 ? liveMatches : (apiLive ?? []);

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold mb-2">Matchs en direct</h1>
          <p className="text-gray-400">Predictions recalculees en temps reel</p>
        </div>
        <div className="flex items-center gap-2">
          <span className={`w-3 h-3 rounded-full ${connected ? 'bg-green-500 animate-pulse' : 'bg-gray-500'}`} />
          <span className="text-sm text-gray-400">
            {connected ? 'WebSocket connecte' : 'Deconnecte'}
          </span>
        </div>
      </div>

      {matches.length === 0 && (
        <div className="card text-center py-12">
          <p className="text-gray-400">Aucun match en direct actuellement.</p>
        </div>
      )}

      <div className="grid gap-4">
        {matches.map((match) => {
          const pred = predictions[match.id];
          return (
            <div key={match.id} className="card">
              <div className="flex items-center justify-between mb-3">
                <span className="text-xs text-red-400 font-medium animate-pulse">
                  LIVE {match.minute ?? 0}'
                </span>
                <span className="text-xs text-gray-400">{match.league_name}</span>
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
              {pred && (
                <div className="mt-4 pt-4 border-t border-pitch-700 flex flex-wrap gap-2">
                  {Object.entries(pred.predictions).slice(0, 4).map(([m, p]) => (
                    <span key={m} className="text-xs bg-pitch-900 px-2 py-1 rounded capitalize">
                      {m.replace(/_/g, ' ')}: {formatProbability(p as number)}
                    </span>
                  ))}
                </div>
              )}
              <Link
                to={`/match/${match.id}`}
                className="text-sm text-accent hover:underline mt-3 inline-block"
              >
                Voir le detail
              </Link>
            </div>
          );
        })}
      </div>
    </div>
  );
}
