import { useParams } from 'react-router-dom';
import { useQuery, useMutation } from '@tanstack/react-query';
import { api, confidenceBadge, formatProbability } from '../api';
import MatchTeams from '../components/MatchTeams';

export default function MatchDetail() {
  const { id } = useParams<{ id: string }>();

  const { data: match, isLoading } = useQuery({
    queryKey: ['match', id],
    queryFn: () => api.getMatch(id!),
    enabled: !!id,
  });

  const { data: stats } = useQuery({
    queryKey: ['match-stats', id],
    queryFn: () => api.getMatchStats(id!),
    enabled: !!id,
  });

  const { data: prediction, refetch } = useQuery({
    queryKey: ['prediction', id],
    queryFn: () => api.getPrediction(id!),
    enabled: !!id,
  });

  const analyze = useMutation({
    mutationFn: () => api.analyzeMatch(id!),
    onSuccess: () => refetch(),
  });

  const analyzeError = analyze.error instanceof Error ? analyze.error.message : null;

  if (isLoading) return <p className="text-gray-400">Chargement...</p>;
  if (!match) return <p className="text-red-400">Match introuvable</p>;

  return (
    <div className="max-w-4xl mx-auto">
      <div className="card mb-6">
        <p className="text-sm text-gray-400 mb-2">{match.league_name} - {match.round}</p>
        <MatchTeams
          home={match.home_team}
          away={match.away_team}
          homeScore={match.home_score}
          awayScore={match.away_score}
          minute={match.minute}
          status={match.status}
          layout="detail"
        />
        <p className="text-sm text-gray-400 text-center">
          {new Date(match.kickoff_at).toLocaleString('fr-FR')}
          {match.venue && ` - ${match.venue}`}
        </p>
      </div>

      {stats && Array.isArray(stats) && stats.length > 0 && (
        <div className="card mb-6">
          <h3 className="text-lg font-semibold mb-4">Statistiques</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-gray-400">
                  <th className="text-left py-2">Stat</th>
                  <th className="text-center py-2">{match.home_team.name}</th>
                  <th className="text-center py-2">{match.away_team.name}</th>
                </tr>
              </thead>
              <tbody>
                {['total_shots', 'shots_on_goal', 'corner_kicks', 'ball_possession', 'expected_goals'].map(
                  (stat) => {
                    const homeStat = (stats as Record<string, unknown>[]).find(
                      (s) => s.team_name === match.home_team.name
                    );
                    const awayStat = (stats as Record<string, unknown>[]).find(
                      (s) => s.team_name === match.away_team.name
                    );
                    return (
                      <tr key={stat} className="border-t border-pitch-700">
                        <td className="py-2 capitalize">{stat.replace(/_/g, ' ')}</td>
                        <td className="text-center py-2">{String(homeStat?.[stat] ?? '-')}</td>
                        <td className="text-center py-2">{String(awayStat?.[stat] ?? '-')}</td>
                      </tr>
                    );
                  }
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <div className="card mb-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold">Predictions ML</h3>
          <button
            onClick={() => analyze.mutate()}
            disabled={analyze.isPending}
            className="btn-primary text-sm"
          >
            {analyze.isPending ? 'Analyse...' : 'Analyser avec IA'}
          </button>
        </div>

        {analyzeError && (
          <div className="bg-red-900/30 border border-red-700 rounded-lg p-3 mb-4 text-sm text-red-300">
            Erreur lors de l'analyse : {analyzeError}
          </div>
        )}

        {prediction?.no_bet_recommended && (
          <div className="bg-gray-800/50 rounded-lg p-4 mb-4">
            <span className="badge-abstain text-sm">Aucun pronostic recommande</span>
            <p className="text-gray-400 text-sm mt-2">
              Confiance insuffisante pour recommander un pari.
            </p>
          </div>
        )}

        {prediction && (
          <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
            {Object.entries(prediction.predictions).map(([market, prob]) => (
              <div key={market} className="bg-pitch-900 rounded-lg p-3">
                <p className="text-xs text-gray-400 capitalize mb-1">
                  {market.replace(/_/g, ' ')}
                </p>
                <p className="text-xl font-bold">{formatProbability(prob)}</p>
                {prediction.confidence[market] && (
                  <span className={`${confidenceBadge(prediction.confidence[market])} mt-1`}>
                    Confiance {formatProbability(prediction.confidence[market])}
                  </span>
                )}
              </div>
            ))}
          </div>
        )}

        <p className="text-xs text-gray-500 mt-4">
          Estimations statistiques du modele ML. Aucune garantie de gain.
        </p>
      </div>

      {prediction?.ai_analysis && (
        <div className="card">
          <h3 className="text-lg font-semibold mb-3">Analyse IA</h3>
          <p className="text-gray-300 whitespace-pre-line leading-relaxed">
            {prediction.ai_analysis}
          </p>
          {prediction.ai_reasons && prediction.ai_reasons.length > 0 && (
            <div className="mt-4">
              <h4 className="text-sm font-medium text-gray-400 mb-2">Facteurs cles</h4>
              <ul className="space-y-1">
                {(prediction.ai_reasons as string[]).map((reason, i) => (
                  <li key={i} className="flex items-start gap-2 text-sm">
                    <span className="text-accent mt-0.5">&#10003;</span>
                    {reason}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
