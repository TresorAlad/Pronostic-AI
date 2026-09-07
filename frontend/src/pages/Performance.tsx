import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  BarChart,
  Bar,
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
  Legend,
} from 'recharts';
import { api, formatProbability } from '../api';

export default function Performance() {
  const queryClient = useQueryClient();

  const { data: performance, isLoading, isError, error, refetch } = useQuery({
    queryKey: ['performance'],
    queryFn: api.getPerformance,
  });

  const { data: trend } = useQuery({
    queryKey: ['performance-trend'],
    queryFn: api.getPerformanceTrend,
  });

  const runEval = useMutation({
    mutationFn: api.runEvaluation,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['performance'] });
      queryClient.invalidateQueries({ queryKey: ['performance-trend'] });
    },
  });

  const chartData = performance?.map((p) => ({
    name: `${p.model_name} - ${p.market}`,
    accuracy: p.accuracy ? Math.round(p.accuracy * 100) : 0,
    brier: p.brier_score ? Math.round(p.brier_score * 1000) / 1000 : 0,
  })) ?? [];

  const trendByPeriod = (() => {
    const map = new Map<string, Record<string, number | string>>();
    for (const t of trend ?? []) {
      const row = map.get(t.period) ?? { period: t.period };
      row[t.market] = Math.round(t.accuracy * 100);
      map.set(t.period, row);
    }
    return Array.from(map.values()).reverse();
  })();

  const trendMarkets = Array.from(new Set((trend ?? []).map((t) => t.market)));

  const errorMessage = isError && error instanceof Error ? error.message : null;
  const evalError = runEval.error instanceof Error ? runEval.error.message : null;

  return (
    <div>
      <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 className="page-title">Performance</h1>
          <p className="page-subtitle">Précision des prédictions comparée aux résultats réels</p>
        </div>
        <div className="flex gap-2">
          <button onClick={() => refetch()} className="btn-secondary shrink-0 text-sm">
            Actualiser
          </button>
          <button
            onClick={() => runEval.mutate()}
            disabled={runEval.isPending}
            className="btn-primary shrink-0 text-sm"
          >
            {runEval.isPending ? 'Évaluation...' : 'Évaluer'}
          </button>
        </div>
      </div>

      {isLoading && <p className="text-slate-400">Chargement...</p>}

      {errorMessage && (
        <div className="card mb-6 border-red-500/30 bg-red-500/5">
          <p className="text-red-300 text-sm">{errorMessage}</p>
        </div>
      )}

      {evalError && (
        <div className="card mb-6 border-red-500/30 bg-red-500/5">
          <p className="text-red-300 text-sm">{evalError}</p>
        </div>
      )}

      {!isLoading && (!performance || performance.length === 0) && (
        <div className="card mb-6 text-center py-12">
          <p className="text-slate-400 mb-2">Aucune métrique disponible pour le moment.</p>
          <p className="text-sm text-slate-500 max-w-md mx-auto">
            Lancez l&apos;évaluation post-match pour comparer les prédictions aux résultats réels et
            remplir les courbes de performance.
          </p>
          <button
            onClick={() => runEval.mutate()}
            disabled={runEval.isPending}
            className="btn-primary mt-6 text-sm"
          >
            {runEval.isPending ? 'Évaluation en cours...' : 'Lancer l\'évaluation'}
          </button>
        </div>
      )}

      {trendByPeriod.length > 0 && (
        <div className="card mb-6">
          <h3 className="font-display text-lg font-semibold text-heading mb-4">
            Précision dans le temps (par semaine)
          </h3>
          <div className="w-full h-[280px]">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={trendByPeriod}>
                <CartesianGrid strokeDasharray="3 3" stroke="#cbd5e1" />
                <XAxis dataKey="period" tick={{ fill: '#64748b', fontSize: 11 }} />
                <YAxis tick={{ fill: '#64748b' }} domain={[0, 100]} />
                <Tooltip />
                <Legend />
                {trendMarkets.slice(0, 5).map((market, i) => (
                  <Line
                    key={market}
                    type="monotone"
                    dataKey={market}
                    stroke={['#10b981', '#3b82f6', '#f59e0b', '#8b5cf6', '#ef4444'][i % 5]}
                    dot={false}
                  />
                ))}
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}

      {chartData.length > 0 && (
        <div className="card mb-6">
          <h3 className="font-display text-lg font-semibold text-heading mb-4">
            Précision par marché
          </h3>
          <div className="w-full h-[300px]">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#cbd5e1" />
                <XAxis dataKey="name" tick={{ fill: '#64748b', fontSize: 11 }} angle={-20} />
                <YAxis tick={{ fill: '#64748b' }} domain={[0, 100]} />
                <Tooltip />
                <Bar dataKey="accuracy" fill="#10b981" radius={[6, 6, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}

      {performance && performance.length > 0 && (
        <div className="card overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-slate-400 border-b border-navy-600">
                <th className="text-left py-3 font-medium">Source</th>
                <th className="text-left py-3 font-medium">Marché</th>
                <th className="text-right py-3 font-medium">Précision</th>
                <th className="text-right py-3 font-medium">Fiabilité</th>
                <th className="text-right py-3 font-medium">Échantillon</th>
              </tr>
            </thead>
            <tbody>
              {performance.map((p, i) => (
                <tr key={i} className="border-b border-navy-600/50 hover:bg-navy-800/40 transition-colors">
                  <td className="py-3 text-slate-800 dark:text-slate-200">
                    {p.model_name}
                  </td>
                  <td className="py-3 capitalize text-slate-600 dark:text-slate-300">
                    {p.market.replace(/_/g, ' ')}
                  </td>
                  <td className="text-right py-3 text-brand-dark dark:text-brand-light">
                    {p.accuracy != null ? formatProbability(p.accuracy) : '-'}
                  </td>
                  <td className="text-right py-3 text-slate-400">{p.brier_score?.toFixed(4) ?? '-'}</td>
                  <td className="text-right py-3 text-slate-400">{p.sample_size ?? '-'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
