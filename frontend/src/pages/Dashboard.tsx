import { useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { api, confidenceBadge, formatProbability } from '../api';
import MatchTeams from '../components/MatchTeams';
import FilterSelect, { FilterDate } from '../components/FilterSelect';
import { useMatchStatusWebSocket } from '../hooks/useMatchStatusWebSocket';

type StatusFilter = 'all' | 'scheduled' | 'live' | 'finished';

const STATUS_OPTIONS = [
  { value: 'all', label: 'Tous' },
  { value: 'scheduled', label: 'À venir', hint: 'scheduled' as const },
  { value: 'live', label: 'Live', hint: 'live' as const },
  { value: 'finished', label: 'Terminés', hint: 'finished' as const },
];

export default function Dashboard() {
  const queryClient = useQueryClient();
  const [leagueFilter, setLeagueFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('scheduled');
  const [dateFilter, setDateFilter] = useState('');

  const { data: matches, isLoading, isError, error, refetch, isFetching } = useQuery({
    queryKey: ['matches-today', statusFilter === 'scheduled'],
    queryFn: () => api.getMatchesToday(statusFilter === 'scheduled'),
    refetchInterval: 60000,
  });

  const { data: liveMatches } = useQuery({
    queryKey: ['live-matches-count'],
    queryFn: api.getLiveMatches,
    refetchInterval: 30000,
  });

  useMatchStatusWebSocket((event) => {
    if (event.to === 'live') {
      queryClient.invalidateQueries({ queryKey: ['matches-today'] });
      queryClient.invalidateQueries({ queryKey: ['live-matches-count'] });
    }
  });

  const hasMatches = (matches?.length ?? 0) > 0;
  const backendDown = isError && !hasMatches;
  const refreshFailed = isError && hasMatches;

  const leagues = useMemo(() => {
    const names = new Set(matches?.map((m) => m.league_name) ?? []);
    return Array.from(names).sort();
  }, [matches]);

  const filtered = useMemo(() => {
    return (matches ?? []).filter((m) => {
      if (leagueFilter !== 'all' && m.league_name !== leagueFilter) return false;
      if (statusFilter !== 'all' && m.status !== statusFilter) return false;
      if (dateFilter && !m.kickoff_at.startsWith(dateFilter)) return false;
      return true;
    });
  }, [matches, leagueFilter, statusFilter, dateFilter]);

  const leagueOptions = useMemo(
    () => [{ value: 'all', label: 'Toutes les ligues' }, ...leagues.map((l) => ({ value: l, label: l }))],
    [leagues]
  );

  const activeFilters =
    (leagueFilter !== 'all' ? 1 : 0) + (statusFilter !== 'all' ? 1 : 0) + (dateFilter ? 1 : 0);

  const resetFilters = () => {
    setLeagueFilter('all');
    setStatusFilter('all');
    setDateFilter('');
  };

  const hasScheduled = filtered.some((m) => m.status === 'scheduled');
  const showRecentFallback = filtered.length > 0 && !hasScheduled;

  return (
    <div>
      <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 className="page-title">Matchs du jour</h1>
          <p className="page-subtitle">
            Top 5 européen · Prédictions basées sur le machine learning et les stats réelles
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
          label="Ligue"
          value={leagueFilter}
          options={leagueOptions}
          onChange={setLeagueFilter}
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
          <p className="text-red-300 font-medium">Backend inaccessible</p>
          <p className="text-sm text-slate-500 mt-2">
            Lancez le backend :{' '}
            <code className="text-brand-dark dark:text-brand-light">cd backend && go run ./cmd/server</code>
          </p>
          <p className="text-xs text-slate-600 mt-1">{(error as Error).message}</p>
        </div>
      )}

      {refreshFailed && (
        <div className="card mb-4 border-amber-500/30 bg-amber-500/5 px-4 py-3">
          <p className="text-amber-800 dark:text-amber-200 text-sm">
            Impossible de rafraîchir les matchs. Affichage des dernières données connues.
          </p>
          <p className="text-xs text-slate-500 mt-1">{(error as Error).message}</p>
        </div>
      )}

      {hasMatches && showRecentFallback && (
        <p className="text-sm text-gold-light/90 mb-4 rounded-xl border border-gold/20 bg-gold/5 px-4 py-3">
          Aucun match Top 5 à venir. Derniers résultats des grands championnats.
        </p>
      )}

      {hasMatches && (
        <div className="grid gap-4">
          {filtered.length === 0 && (
            <div className="card text-center py-12">
              <p className="text-slate-400">Aucun match pour ces filtres.</p>
            </div>
          )}

          {filtered.map((match) => (
            <MatchCard key={match.id} match={match} />
          ))}
        </div>
      )}

      {!isLoading && !backendDown && !hasMatches && (
        <div className="card text-center py-12">
          <p className="text-slate-400">Aucun match disponible.</p>
          <p className="text-sm text-slate-500 mt-2">
            Lancez :{' '}
            <code className="text-brand-dark dark:text-brand-light">
              go run ./cmd/collector -mode=sync-today
            </code>
          </p>
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
