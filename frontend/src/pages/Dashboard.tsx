import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { api, confidenceBadge, formatProbability } from '../api';
import MatchTeams from '../components/MatchTeams';

export default function Dashboard() {
  const { data: matches, isLoading, isError, error, refetch } = useQuery({
    queryKey: ['matches-today'],
    queryFn: api.getMatchesToday,
    refetchInterval: 60000,
  });

  const hasScheduled = matches?.some((m) => m.status === 'scheduled');
  const showRecentFallback = matches?.length && !hasScheduled;

  return (
    <div>
      <div className="mb-8 flex items-start justify-between">
        <div>
          <h1 className="text-3xl font-bold mb-2">Matchs du jour</h1>
          <p className="text-gray-400">Top 5 europeen - Pronostics bases sur le Machine Learning</p>
        </div>
        <button onClick={() => refetch()} className="btn-primary text-sm">
          Actualiser
        </button>
      </div>

      {isLoading && <p className="text-gray-400">Chargement...</p>}

      {isError && (
        <div className="card text-center py-12 border-red-900/50">
          <p className="text-red-400">Backend inaccessible</p>
          <p className="text-sm text-gray-500 mt-2">
            Lancez le backend : <code className="text-accent">cd backend && go run ./cmd/server</code>
          </p>
          <p className="text-xs text-gray-600 mt-1">{(error as Error).message}</p>
        </div>
      )}

      {showRecentFallback && (
        <p className="text-sm text-yellow-500/80 mb-4">
          Aucun match Top 5 a venir. Derniers resultats des grands championnats.
        </p>
      )}

      <div className="grid gap-4">
        {!isLoading && !isError && matches?.length === 0 && (
          <div className="card text-center py-12">
            <p className="text-gray-400">Aucun match disponible.</p>
            <p className="text-sm text-gray-500 mt-2">
              Lancez : <code className="text-accent">go run ./cmd/collector -mode=sync-today</code>
            </p>
          </div>
        )}

        {matches?.map((match) => (
          <MatchCard key={match.id} match={match} />
        ))}
      </div>
    </div>
  );
}

function MatchCard({ match }: { match: import('../api').Match }) {
  const { data: prediction } = useQuery({
    queryKey: ['prediction', match.id],
    queryFn: () => api.getPrediction(match.id),
    retry: 1,
  });

  const kickoff = new Date(match.kickoff_at).toLocaleTimeString('fr-FR', {
    hour: '2-digit',
    minute: '2-digit',
  });

  return (
    <Link to={`/match/${match.id}`} className="card hover:border-accent/50 transition-colors block">
      <div className="flex items-center justify-between mb-3">
        <span className="text-xs text-gray-400 uppercase tracking-wide">{match.league_name}</span>
        <span className="text-xs text-gray-400">{kickoff}</span>
      </div>

      <MatchTeams
        home={match.home_team}
        away={match.away_team}
        homeScore={match.home_score}
        awayScore={match.away_score}
        minute={match.minute}
        status={match.status}
        layout="card"
      />

      {prediction && (
        <div className="mt-4 pt-4 border-t border-pitch-700">
          {prediction.no_bet_recommended ? (
            <span className="badge-abstain">Aucun pronostic recommande</span>
          ) : (
            <div className="flex flex-wrap gap-2">
              {Object.entries(prediction.confidence)
                .slice(0, 4)
                .map(([market, conf]) => (
                  <span key={market} className={confidenceBadge(conf)}>
                    {market.replace(/_/g, ' ')} {formatProbability(conf)}
                  </span>
                ))}
            </div>
          )}
        </div>
      )}
    </Link>
  );
}
