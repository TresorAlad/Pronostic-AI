import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { api, confidenceBadge, formatProbability } from '../api';
import { useAuth } from '../hooks/useAuth';
import { marketLabel } from '../utils/marketLabels';

interface SavedCoupon {
  id: string;
  name: string;
  created_at: string;
  selection_count: number;
}

export default function MyCoupons() {
  const { isAuthenticated, loading } = useAuth();
  const { data: coupons, isLoading, isError, error } = useQuery({
    queryKey: ['my-coupons'],
    queryFn: () => api.getMyCoupons(),
    enabled: isAuthenticated,
    retry: false,
  });

  if (loading) {
    return <p className="text-slate-400">Chargement...</p>;
  }

  if (!isAuthenticated) {
    return (
      <div className="card text-center py-12">
        <p className="text-slate-400">Connectez-vous pour voir vos coupons sauvegardés.</p>
        <Link to="/auth" className="btn-primary mt-4 inline-block">
          Se connecter
        </Link>
      </div>
    );
  }

  if (isError) {
    const msg = error instanceof Error ? error.message : 'Erreur';
    return (
      <div className="card text-center py-12">
        <p className="text-red-300 text-sm">{msg}</p>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-8">
        <h1 className="page-title">Mes coupons</h1>
        <p className="page-subtitle">Historique de vos coupons générés et sauvegardés.</p>
      </div>

      {isLoading && <p className="text-slate-400">Chargement...</p>}

      {!isLoading && (!coupons || coupons.length === 0) && (
        <div className="card text-center py-12">
          <p className="text-slate-400">Aucun coupon sauvegardé.</p>
          <Link to="/coupon" className="btn-primary mt-4 inline-block">
            Générer un coupon
          </Link>
        </div>
      )}

      <div className="grid gap-4">
        {coupons?.map((c) => (
          <CouponSummary key={c.id} coupon={c} />
        ))}
      </div>
    </div>
  );
}

function CouponSummary({ coupon }: { coupon: SavedCoupon }) {
  const { data: detail } = useQuery({
    queryKey: ['coupon', coupon.id],
    queryFn: () => api.getMyCoupon(coupon.id),
  });

  const date = new Date(coupon.created_at).toLocaleString('fr-FR');
  const selections = (detail?.selections as Array<Record<string, unknown>>) ?? [];

  return (
    <div className="card">
      <div className="flex items-start justify-between gap-4 mb-4">
        <div>
          <h3 className="font-display text-lg font-semibold text-heading">{coupon.name}</h3>
          <p className="text-xs text-slate-500 mt-1">{date}</p>
        </div>
        <span className="badge-medium">{coupon.selection_count} sélections</span>
      </div>
      {selections.length > 0 && (
        <ul className="space-y-2 text-sm">
          {selections.slice(0, 5).map((sel, i) => (
            <li key={i} className="flex justify-between gap-2 border-t border-navy-600/40 pt-2">
              <span className="text-slate-600 dark:text-slate-300 truncate">
                {String(sel.home_team)} vs {String(sel.away_team)} ·{' '}
                {marketLabel(String(sel.selection ?? sel.market))}
              </span>
              <span className={confidenceBadge(Number(sel.confidence ?? 0))}>
                {formatProbability(Number(sel.confidence ?? 0))}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
