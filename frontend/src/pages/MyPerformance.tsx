import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from 'recharts';
import { api, formatProbability } from '../api';
import { useAuth } from '../hooks/useAuth';
import { marketLabel } from '../utils/marketLabels';

export default function MyPerformance() {
  const { isAuthenticated, loading } = useAuth();
  const { data, isLoading } = useQuery({
    queryKey: ['my-performance'],
    queryFn: api.getMyPerformance,
    enabled: isAuthenticated,
  });

  if (loading || isLoading) {
    return <p className="text-slate-400">Chargement...</p>;
  }

  if (!isAuthenticated) {
    return (
      <div className="card text-center py-12">
        <p className="text-slate-400">Connectez-vous pour voir votre performance.</p>
        <Link to="/auth" className="btn-primary mt-4 inline-block">
          Se connecter
        </Link>
      </div>
    );
  }

  const summary = data?.summary;
  const trend = [...(data?.trend ?? [])].reverse();
  const recent = data?.recent ?? [];

  return (
    <div className="max-w-4xl mx-auto">
      <div className="mb-8">
        <h1 className="page-title">Ma performance</h1>
        <p className="page-subtitle">Suivi de vos sélections de coupons après les matchs terminés.</p>
      </div>

      {!summary || summary.total_selections === 0 ? (
        <div className="card text-center py-12">
          <p className="text-slate-400">Aucune sélection évaluée pour le moment.</p>
          <p className="text-sm text-slate-500 mt-2">
            Générez un coupon, puis attendez la fin des matchs et l&apos;évaluation post-match.
          </p>
          <Link to="/coupon" className="btn-primary mt-4 inline-block">
            Générer un coupon
          </Link>
        </div>
      ) : (
        <>
          <div className="grid gap-4 sm:grid-cols-3 mb-6">
            <StatCard label="Sélections évaluées" value={String(summary.total_selections)} />
            <StatCard label="Correctes" value={String(summary.correct_count)} />
            <StatCard label="Précision" value={formatProbability(summary.accuracy)} />
          </div>

          {trend.length > 0 && (
            <div className="card mb-6">
              <h3 className="font-display text-lg font-semibold text-heading mb-4">Par semaine</h3>
              <div className="h-[240px]">
                <ResponsiveContainer width="100%" height="100%">
                  <LineChart data={trend.map((t) => ({ ...t, accuracy_pct: Math.round(t.accuracy * 100) }))}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#cbd5e1" />
                    <XAxis dataKey="period" tick={{ fontSize: 11 }} />
                    <YAxis domain={[0, 100]} />
                    <Tooltip />
                    <Line type="monotone" dataKey="accuracy_pct" stroke="#10b981" dot={false} />
                  </LineChart>
                </ResponsiveContainer>
              </div>
            </div>
          )}

          <div className="card">
            <h3 className="font-display text-lg font-semibold text-heading mb-4">Dernières sélections</h3>
            <ul className="space-y-2 text-sm">
              {recent.map((row, i) => (
                <li key={i} className="flex justify-between gap-3 border-t border-slate-200 dark:border-navy-600 pt-2">
                  <span className="text-slate-600 dark:text-slate-300 truncate">
                    {String(row.home_team)} vs {String(row.away_team)} · {marketLabel(String(row.selection ?? row.market))}
                  </span>
                  <span className={row.is_correct ? 'text-emerald-500' : 'text-red-400'}>
                    {row.is_correct ? 'Correct' : 'Incorrect'}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        </>
      )}
    </div>
  );
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="card text-center">
      <p className="text-2xl font-display font-bold text-brand-dark dark:text-brand-light">{value}</p>
      <p className="text-sm text-slate-500 mt-1">{label}</p>
    </div>
  );
}
