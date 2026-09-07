import { useQuery } from '@tanstack/react-query';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from 'recharts';
import { api, formatProbability } from '../api';

export default function Performance() {
  const { data: performance, isLoading } = useQuery({
    queryKey: ['performance'],
    queryFn: api.getPerformance,
  });

  const chartData = performance?.map((p) => ({
    name: `${p.model_name} - ${p.market}`,
    accuracy: p.accuracy ? Math.round(p.accuracy * 100) : 0,
    brier: p.brier_score ? Math.round(p.brier_score * 1000) / 1000 : 0,
  })) ?? [];

  return (
    <div>
      <h1 className="text-3xl font-bold mb-2">Performance des modeles</h1>
      <p className="text-gray-400 mb-8">Metriques de backtesting et evaluation post-match</p>

      {isLoading && <p className="text-gray-400">Chargement...</p>}

      {!isLoading && (!performance || performance.length === 0) && (
        <div className="card text-center py-12">
          <p className="text-gray-400">Aucune metrique disponible.</p>
          <p className="text-sm text-gray-500 mt-2">
            Les metriques apparaitront apres l'evaluation post-match.
          </p>
        </div>
      )}

      {chartData.length > 0 && (
        <div className="card mb-6">
          <h3 className="text-lg font-semibold mb-4">Accuracy par marche</h3>
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#1a3a1a" />
              <XAxis dataKey="name" tick={{ fill: '#9ca3af', fontSize: 11 }} angle={-20} />
              <YAxis tick={{ fill: '#9ca3af' }} domain={[0, 100]} />
              <Tooltip
                contentStyle={{ background: '#122812', border: '1px solid #1a3a1a' }}
              />
              <Bar dataKey="accuracy" fill="#22c55e" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
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
                    {p.accuracy ? formatProbability(p.accuracy) : '-'}
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
