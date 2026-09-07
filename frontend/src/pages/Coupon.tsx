import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useMutation } from '@tanstack/react-query';
import { api, confidenceBadge, formatProbability, type CouponSelection } from '../api';
import { useAuth } from '../hooks/useAuth';
import { marketCategory, marketCategoryClass, marketLabel } from '../utils/marketLabels';

const SLIDER_MIN = 50;
const SLIDER_MAX = 90;

function clampConfidencePercent(value: number) {
  return Math.max(SLIDER_MIN, Math.min(SLIDER_MAX, value));
}

export default function Coupon() {
  const navigate = useNavigate();
  const { isAuthenticated, loading, refresh } = useAuth();
  const [minConfidence, setMinConfidence] = useState(0.55);
  const confidencePercent = clampConfidencePercent(Math.round(minConfidence * 100));
  const [coupon, setCoupon] = useState<{
    id: string;
    selections: CouponSelection[];
    disclaimer: string;
  } | null>(null);

  useEffect(() => {
    if (!loading && !isAuthenticated) {
      navigate('/auth', { state: { from: '/coupon' } });
    }
  }, [isAuthenticated, loading, navigate]);

  const generate = useMutation({
    mutationFn: () => api.generateCoupon(minConfidence, 8),
    onSuccess: (data) => {
      setCoupon({
        id: data.id,
        disclaimer: data.disclaimer ?? '',
        selections: Array.isArray(data.selections) ? data.selections : [],
      });
      refresh();
    },
  });

  const errorMessage = generate.error instanceof Error ? generate.error.message : null;

  return (
    <div className="max-w-2xl mx-auto">
      <div className="mb-8">
        <h1 className="page-title">Coupon IA</h1>
        <p className="page-subtitle">
          Coupon type pronostiqueur : une sélection par catégorie (temps réglementaire, buts, BTTS,
          double chance, tirs, corners, cartons, fautes, hors-jeu, possession).
        </p>
      </div>

      <div className="card mb-6">
        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-3">
          Confiance minimum :{' '}
          <span className="text-brand-dark dark:text-brand-light">{formatProbability(minConfidence)}</span>
        </label>
        <input
          type="range"
          min={0}
          max={100}
          step={5}
          value={confidencePercent}
          onChange={(e) =>
            setMinConfidence(clampConfidencePercent(parseInt(e.target.value, 10)) / 100)
          }
          style={{ ['--slider-fill' as string]: `${confidencePercent}%` }}
          className="confidence-slider"
        />
        <div className="flex justify-between text-xs text-slate-500 mt-2 px-0.5">
          <span>50 % (min)</span>
          <span>90 % (max)</span>
        </div>
        <button
          onClick={() => generate.mutate()}
          disabled={generate.isPending}
          className="btn-primary w-full mt-5"
        >
          {generate.isPending ? 'Génération en cours...' : 'Générer le coupon'}
        </button>
        <PrewarmButton />
        {generate.isPending && (
          <p className="text-xs text-slate-500 mt-3 text-center">
            Prédictions mises en cache quand possible (objectif &lt; 5 s)...
          </p>
        )}
      </div>

      {errorMessage && (
        <div className="card mb-6 border-red-500/30 bg-red-500/5">
          <p className="text-red-300 text-sm">{errorMessage}</p>
        </div>
      )}

      {coupon && (
        <div className="card">
          <h3 className="font-display text-lg font-semibold text-heading mb-4">Coupon généré</h3>
          {(coupon.selections?.length ?? 0) === 0 ? (
            <p className="text-slate-400">
              Aucune sélection ne dépasse le seuil ({formatProbability(minConfidence)}).
            </p>
          ) : (
            <ol className="space-y-3">
              {coupon.selections.map((sel, i) => {
                const category = sel.market_category ?? marketCategory(sel.market);
                const label = sel.market_label ?? marketLabel(sel.selection);
                return (
                  <li key={`${sel.match_id}-${sel.market}-${i}`} className="selection-row">
                    <span className="font-display text-brand-dark dark:text-brand-light font-bold text-lg w-6 shrink-0">
                      {i + 1}
                    </span>
                    <div className="flex-1 min-w-0">
                      <p className="font-semibold text-heading">
                        {sel.home_team} vs {sel.away_team}
                      </p>
                      <span
                        className={`inline-block mt-2 text-xs font-semibold px-2.5 py-0.5 rounded-lg border ${marketCategoryClass(category)}`}
                      >
                        {category}
                      </span>
                      <p className="text-sm text-slate-300 mt-2">{label}</p>
                    </div>
                    <span className={`${confidenceBadge(sel.confidence)} shrink-0`}>
                      {formatProbability(sel.confidence)}
                    </span>
                  </li>
                );
              })}
            </ol>
          )}
          <p className="text-xs text-slate-500 mt-5 border-t border-navy-600 pt-4">{coupon.disclaimer}</p>
        </div>
      )}
    </div>
  );
}

function PrewarmButton() {
  const prewarm = useMutation({ mutationFn: api.prewarmPredictions });
  return (
    <button
      type="button"
      onClick={() => prewarm.mutate()}
      disabled={prewarm.isPending}
      className="btn-secondary w-full mt-2 text-sm"
    >
      {prewarm.isPending ? 'Pre-chauffage...' : 'Pre-chauffer le cache'}
    </button>
  );
}
