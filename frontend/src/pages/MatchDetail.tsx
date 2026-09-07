import { useParams } from 'react-router-dom';
import { useQuery, useMutation } from '@tanstack/react-query';
import { api, confidenceBadge, formatProbability } from '../api';
import MatchTeams from '../components/MatchTeams';
import { groupPredictions, marketLabel } from '../utils/marketLabels';

const STAT_ROWS = [
  { key: 'total_shots', label: 'Tirs' },
  { key: 'shots_on_goal', label: 'Tirs cadrés' },
  { key: 'corner_kicks', label: 'Corners' },
  { key: 'ball_possession', label: 'Possession (%)' },
  { key: 'expected_goals', label: 'xG' },
  { key: 'fouls', label: 'Fautes' },
  { key: 'yellow_cards', label: 'Cartons jaunes' },
  { key: 'offsides', label: 'Hors-jeu' },
];

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

  if (isLoading) return <p className="text-slate-400">Chargement...</p>;
  if (!match) return <p className="text-red-400">Match introuvable</p>;

  const predictionGroups = prediction?.predictions
    ? groupPredictions(prediction.predictions)
    : [];

  return (
    <div className="max-w-4xl mx-auto">
      <div className="card mb-6">
        <p className="text-sm text-slate-400 mb-3">
          {match.league_name} · {match.round}
        </p>
        <MatchTeams
          home={match.home_team}
          away={match.away_team}
          homeScore={match.home_score}
          awayScore={match.away_score}
          minute={match.minute}
          status={match.status}
          layout="detail"
        />
        <p className="text-sm text-slate-500 text-center mt-4">
          {new Date(match.kickoff_at).toLocaleString('fr-FR')}
          {match.venue && ` · ${match.venue}`}
        </p>
      </div>

      {stats && Array.isArray(stats) && stats.length > 0 && (
        <div className="card mb-6">
          <h3 className="font-display text-lg font-semibold text-heading mb-4">Statistiques</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-slate-400">
                  <th className="text-left py-2 font-medium">Stat</th>
                  <th className="text-center py-2 font-medium">{match.home_team.name}</th>
                  <th className="text-center py-2 font-medium">{match.away_team.name}</th>
                </tr>
              </thead>
              <tbody>
                {STAT_ROWS.map(({ key, label }) => {
                  const homeStat = (stats as Record<string, unknown>[]).find(
                    (s) => s.team_name === match.home_team.name
                  );
                  const awayStat = (stats as Record<string, unknown>[]).find(
                    (s) => s.team_name === match.away_team.name
                  );
                  return (
                    <tr key={key} className="border-t border-navy-600/80">
                      <td className="py-2.5 text-slate-600 dark:text-slate-300">{label}</td>
                      <td className="text-center py-2.5 font-medium text-heading">
                        {String(homeStat?.[key] ?? '-')}
                      </td>
                      <td className="text-center py-2.5 font-medium text-heading">
                        {String(awayStat?.[key] ?? '-')}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <div className="card mb-6">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between mb-5">
          <h3 className="font-display text-lg font-semibold text-heading">Prédictions ML</h3>
          <button
            onClick={() => analyze.mutate()}
            disabled={analyze.isPending}
            className="btn-primary text-sm shrink-0"
          >
            {analyze.isPending ? 'Analyse...' : 'Analyser avec IA'}
          </button>
        </div>

        {analyzeError && (
          <div className="rounded-xl border border-red-500/30 bg-red-500/5 p-3 mb-4 text-sm text-red-300">
            Erreur lors de l&apos;analyse : {analyzeError}
          </div>
        )}

        {prediction?.no_bet_recommended && (
          <div className="rounded-xl border border-navy-600 bg-navy-900/60 p-4 mb-5">
            <span className="badge-abstain text-sm">Confiance globale faible</span>
            <p className="text-slate-400 text-sm mt-2">
              Aucun marché ne dépasse fortement le seuil, mais les probabilités ci-dessous restent
              disponibles pour comparaison.
            </p>
          </div>
        )}

        {prediction && (
          <div className="space-y-6">
            {predictionGroups.map((group) => (
              <div key={group.title}>
                <h4 className="text-sm font-semibold text-brand-dark dark:text-brand-light mb-3">{group.title}</h4>
                <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
                  {group.items.map(({ key, value }) => (
                    <div
                      key={key}
                      className="rounded-xl border border-slate-200 bg-slate-50 p-3 transition-colors hover:border-brand/30 dark:border-navy-600/60 dark:bg-navy-900/50 dark:hover:border-brand/20"
                    >
                      <p className="text-xs text-slate-400 mb-1">{marketLabel(key)}</p>
                      <p className="font-display text-xl font-bold text-heading">
                        {key.startsWith('predicted_total') ? value.toFixed(1) : formatProbability(value)}
                      </p>
                      {prediction.confidence?.[key] !== undefined && !key.startsWith('predicted_total') && (
                        <span className={`${confidenceBadge(prediction.confidence[key])} mt-2`}>
                          Confiance {formatProbability(prediction.confidence[key])}
                        </span>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}

        <p className="text-xs text-slate-500 mt-5">
          Estimations statistiques du modèle ML. Aucune garantie de gain.
        </p>
      </div>

      {prediction?.ai_analysis && (
        <div className="card border-brand/20 bg-gradient-to-br from-white to-slate-50 dark:from-navy-850 dark:to-navy-900">
          <h3 className="font-display text-lg font-semibold text-heading mb-3">Analyse IA</h3>
          <p className="text-slate-700 dark:text-slate-300 whitespace-pre-line leading-relaxed">
            {prediction.ai_analysis}
          </p>
          {prediction.ai_reasons && prediction.ai_reasons.length > 0 && (
            <div className="mt-5 pt-5 border-t border-navy-600">
              <h4 className="text-sm font-medium text-slate-400 mb-3">Facteurs clés</h4>
              <ul className="space-y-2">
                {(prediction.ai_reasons as string[]).map((reason, i) => (
                  <li key={i} className="flex items-start gap-2 text-sm text-slate-700 dark:text-slate-300">
                    <span className="text-brand mt-0.5 shrink-0">&#10003;</span>
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
