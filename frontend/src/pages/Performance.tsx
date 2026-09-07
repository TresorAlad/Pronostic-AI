import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from 'recharts';
import { api, formatProbability } from '../api';

export default function Performance() {
  const queryClient = useQueryClient();

  const { data: performance, isLoading, isError, error, refetch } = useQuery({
    queryKey: ['performance'],
    queryFn: api.getPerformance,
  });

  const runEval = useMutation({
    mutationFn: api.runEvaluation,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['performance'] }),
  });

  const chartData = performance?.map((p) => ({
    name: `${p.model_name} - ${p.market}`,
    accuracy: p.accuracy ? Math.round(p.accuracy * 100) : 0,
    brier: p.brier_score ? Math.round(p.brier_score * 1000) / 1000 : 0,
  })) ?? [];

  const errorMessage = isError && error instanceof Error ? error.message : null;
  const evalError = runEval.error instanceof Error ? runEval.error.message : null;

  return (
    <div>
      <div className="flex items-start justify-between mb-8 gap-4">
        <div>
          <h1 className="text-3xl font-bold mb-2">Performance des modeles</h1>
          <p className="text-gray-400">Metriques de validation ML et evaluation post-match</p>
        </div>
        <button onClick={() => refetch()} className="btn-primary text-sm shrink-0">
          Actualiser
        </button>
      </div>

      {isLoading && <p className="text-gray-400">Chargement...</p>}

      {errorMessage && (
        <div className="card mb-6 border-red-900/50 bg-red-900/20">
          <p className="text-red-300 text-sm">{errorMessage}</p>
        </div>
      )}

      {!isLoading && (!performance || performance.length === 0) && (
        <div className="card text-center py-12 mb-6">
          <p className="text-gray-400">Aucune metrique disponible.</p>
          <p className="text-sm text-gray-500 mt-2 mb-4">
            Lancez l'entrainement ML pour generer les metriques de validation :
          </p>
          <code className="text-accent text-sm block mb-4">
            cd ml-pipeline && python train.py
          </code>
          <button
            onClick={() => runEval.mutate()}
            disabled={runEval.isPending}
            className="btn-primary text-sm"
          >
            {runEval.isPending ? 'Evaluation...' : 'Evaluer les predictions terminees'}
          </button>
        </div>
      )}

      {evalError && (
        <div className="card mb-6 border-red-900/50 bg-red-900/20">
          <p className="text-red-300 text-sm">{evalError}</p>
        </div>
      )}

      {chartData.length > 0 && (
        <div className="card mb-6">
          <h3 className="text-lg font-semibold mb-4">Accuracy par marche (validation)</h3>
          <div className="w-full h-[300px]">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#1a3a1a" />
                <XAxis dataKey="name" tick={{ fill: '#9ca3af', fontSize: 11 }} angle={-20} />
                <YAxis tick={{ fill: '#9ca3af' }} domain={[0, 100]} />
                <Tooltip contentStyle={{ background: '#122812', border: '1px solid #1a3a1a' }} />
                <Bar dataKey="accuracy" fill="#22c55e" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}

      {performance && performance.length > 0 && (
        <div className="card">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-gray-400 border-b border-pitch-700">
                <th className="text-left py-3">Modele</th>
                <th className="text-left py-3">Marche</th>
                <th className="text-right py-3">Accuracy</th>
                <th className="text-right py-3">Log Loss</th>
                <th className="text-right py-3">Brier</th>
                <th className="text-right py-3">Echantillon</th>
              </tr>
            </thead>
            <tbody>
              {performance.map((p, i) => (
                <tr key={i} className="border-b border-pitch-700/50">
                  <td className="py-3">{p.model_name} {p.model_version}</td>
                  <td className="py-3 capitalize">{p.market.replace(/_/g, ' ')}</td>
                  <td className="text-right py-3">
                    {p.accuracy != null ? formatProbability(p.accuracy) : '-'}
                  </td>
                  <td className="text-right py-3">{p.log_loss?.toFixed(4) ?? '-'}</td>
                  <td className="text-right py-3">{p.brier_score?.toFixed(4) ?? '-'}</td>
                  <td className="text-right py-3">{p.sample_size ?? '-'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
