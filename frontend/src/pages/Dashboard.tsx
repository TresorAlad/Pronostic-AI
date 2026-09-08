import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { api, confidenceBadge, formatProbability } from '../api';
import MatchTeams from '../components/MatchTeams';
import FilterSelect, { FilterDate } from '../components/FilterSelect';
import { useMatchStatusWebSocket } from '../hooks/useMatchStatusWebSocket';
import { formatDisplayDate, todayLocalISO } from '../utils/date';

type StatusFilter = 'all' | 'scheduled' | 'live' | 'finished';

const STATUS_OPTIONS = [
  { value: 'all', label: 'À venir + Live' },
  { value: 'scheduled', label: 'À venir', hint: 'scheduled' as const },
  { value: 'live', label: 'Live', hint: 'live' as const },
  { value: 'finished', label: 'Terminés', hint: 'finished' as const },
];

function parseTypeFilter(value: string): { leagueId?: string; leagueExternalId?: number } {
  if (value === 'all') return {};
  if (value.startsWith('ext:')) {
    const ext = Number(value.slice(4));
    return Number.isFinite(ext) ? { leagueExternalId: ext } : {};
  }
  return { leagueId: value };
}

export default function Dashboard() {
  const queryClient = useQueryClient();
  const [typeFilter, setTypeFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('scheduled');
  const [dateFilter, setDateFilter] = useState(todayLocalISO());

  const { data: matches, isLoading, isError, error, refetch, isFetching } = useQuery({
    queryKey: ['matches-today', dateFilter, typeFilter, statusFilter],
    queryFn: () => {
      const league = parseTypeFilter(typeFilter);
      return api.getMatchesToday({
        date: dateFilter,
        ...league,
        status: statusFilter,
      });
    },
    refetchInterval: 60000,
  });

  const { data: leagueFilterOptions } = useQuery({
    queryKey: ['leagues-filter-options', dateFilter],
    queryFn: () => api.getActiveLeagues(dateFilter),
  });

  const { data: liveMatches } = useQuery({
    queryKey: ['live-matches-count'],
    queryFn: api.getLiveMatches,
    refetchInterval: 30000,
  });

  useMatchStatusWebSocket((event) => {
    if (event.to === 'live' || event.to === 'finished') {
      queryClient.invalidateQueries({ queryKey: ['matches-today'] });
      queryClient.invalidateQueries({ queryKey: ['leagues-filter-options'] });
      queryClient.invalidateQueries({ queryKey: ['live-matches-count'] });
    }
  });

  const hasMatches = (matches?.length ?? 0) > 0;
  const backendDown = isError && !hasMatches;
  const refreshFailed = isError && hasMatches;

  const typeOptions = [
    { value: 'all', label: 'Tous les championnats' },
    ...(leagueFilterOptions ?? []).map((l) => ({
      value: l.id || `ext:${l.external_id}`,
      label: l.match_count > 0 ? `${l.label} (${l.match_count})` : l.label,
    })),
  ];

  const defaultStatus: StatusFilter = 'scheduled';
  const activeFilters =
    (typeFilter !== 'all' ? 1 : 0) +
    (statusFilter !== defaultStatus && statusFilter !== 'all' ? 1 : 0) +
    (dateFilter !== todayLocalISO() ? 1 : 0);

  const resetFilters = () => {
    setTypeFilter('all');
    setStatusFilter('scheduled');
    setDateFilter(todayLocalISO());
  };

  const hasActiveFilters =
    typeFilter !== 'all' ||
    (statusFilter !== 'scheduled' && statusFilter !== 'all') ||
    dateFilter !== todayLocalISO();

  const matchCount = matches?.length ?? 0;
  const isToday = dateFilter === todayLocalISO();

  return (
    <div>
      <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 className="page-title">Matchs du jour</h1>
          <p className="page-subtitle">
            {isToday
              ? "Compétitions prioritaires : LDC, LE, Top 5 européen — puis secondaires"
              : `Matchs du ${formatDisplayDate(dateFilter)} (triés par importance compétition)`}
          </p>
          <p className="text-sm text-slate-500 mt-1">
            {matchCount} match{matchCount > 1 ? 's' : ''} affiché{matchCount > 1 ? 's' : ''}
          </p>
          {(liveMatches?.length ?? 0) > 0 && (
            <Link to="/live" className="text-sm text-brand-dark dark:text-brand-light mt-2 inline-block">
              {liveMatches!.length} match(s) en live →
            </Link>
          )}
        </div>
        <button onClick={() => refetch()} disabled={isFetching} className="btn-secondary shrink-0 text-sm">
          {isFetching ? 'Actualisation...' : 'Actualiser'}
        </button>
      </div>

      <div className="filter-bar mb-6">
        <div className="flex items-end justify-between gap-3 sm:col-span-3">
          <p className="filter-label mb-0 normal-case tracking-normal text-sm font-medium text-slate-600 dark:text-slate-300">
            Filtrer les matchs
            {activeFilters > 0 && (
              <span className="ml-2 inline-flex rounded-full bg-emerald-500/15 px-2 py-0.5 text-xs font-semibold text-brand-dark dark:text-brand-light">
                {activeFilters} actif{activeFilters > 1 ? 's' : ''}
              </span>
            )}
          </p>
          {activeFilters > 0 && (
            <button type="button" onClick={resetFilters} className="text-xs font-medium text-brand-dark dark:text-brand-light hover:underline">
              Réinitialiser
            </button>
          )}
        </div>

        <FilterSelect
          label="Championnat"
          value={typeFilter}
          options={typeOptions}
          onChange={setTypeFilter}
          placeholder="Tous les championnats"
        />
        <FilterSelect
          label="Statut"
          value={statusFilter}
          options={STATUS_OPTIONS}
          onChange={(v) => setStatusFilter(v as StatusFilter)}
        />
        <FilterDate label="Date" value={dateFilter} onChange={setDateFilter} />
      </div>

      {isLoading && !hasMatches && <p className="text-slate-400">Chargement...</p>}

      {backendDown && (
        <div className="card text-center py-12 border-red-500/30 bg-red-500/5">
          <p className="text-red-300 font-medium">Service momentanément indisponible</p>
          <p className="text-sm text-slate-500 mt-2">
            Réessayez dans quelques instants ou actualisez la page.
          </p>
        </div>
      )}

      {refreshFailed && (
        <div className="card mb-4 border-amber-500/30 bg-amber-500/5 px-4 py-3">
          <p className="text-amber-800 dark:text-amber-200 text-sm">
            Impossible de rafraîchir les matchs. Affichage des dernières données connues.
          </p>
        </div>
      )}

      {hasMatches && (
        <div className="grid gap-4">
          {matches!.map((match) => (
            <MatchCard key={match.id} match={match} />
          ))}
        </div>
      )}

      {!isLoading && !backendDown && !hasMatches && (
        <div className="card text-center py-12">
          <p className="text-slate-400">
            {hasActiveFilters
              ? 'Aucun match pour ces filtres.'
              : "Aucun match prévu ce jour-là pour les championnats suivis."}
          </p>
          {!hasActiveFilters && (
            <p className="text-sm text-slate-500 mt-2">
              Aucune rencontre n&apos;est programmée ce jour-là dans les championnats suivis.
            </p>
          )}
          {hasActiveFilters && (
            <p className="text-sm text-slate-500 mt-2">
              Essayez un autre statut, un autre championnat ou une autre date.
            </p>
          )}
        </div>
      )}
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
    <Link to={`/match/${match.id}`} className="card-hover block">
      <div className="flex items-center justify-between mb-3">
        <span className="text-xs font-semibold uppercase tracking-wider text-brand-dark/90 dark:text-brand-light/80">
          {match.league_name}
        </span>
        <span className="text-xs text-slate-500">{kickoff}</span>
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
        <div className="mt-4 pt-4 border-t border-navy-600/80">
          {prediction.no_bet_recommended ? (
            <span className="badge-abstain">Confiance globale faible</span>
          ) : (
            <div className="flex flex-wrap gap-2">
              {Object.entries(prediction.confidence ?? {})
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
