import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useMutation } from '@tanstack/react-query';
import { api, confidenceBadge, formatOdd, formatProbability, type CouponSelection } from '../api';
import { useAuth } from '../hooks/useAuth';
import { exportCouponPdf } from '../utils/couponPdf';
import { marketCategory, marketCategoryClass, marketLabel } from '../utils/marketLabels';

const SLIDER_MIN = 50;
const SLIDER_MAX = 90;

const ODD_PRESETS = [
  { id: 'prudent', label: 'Prudent', min: 5, max: 10 },
  { id: 'equilibre', label: 'Équilibré', min: 10, max: 25 },
  { id: 'ambitieux', label: 'Ambitieux', min: 25, max: 50 },
] as const;

function clampConfidencePercent(value: number) {
  return Math.max(SLIDER_MIN, Math.min(SLIDER_MAX, value));
}

function clampOdd(value: number) {
  return Math.max(5, Math.min(50, value));
}

function oddTrendLabel(trend?: string) {
  if (trend === 'down') return '▼ Cote en baisse';
  if (trend === 'up') return '▲ Cote en hausse';
  return null;
}

export default function Coupon() {
  const navigate = useNavigate();
  const { isAuthenticated, loading, refresh, user } = useAuth();
  const [preset, setPreset] = useState<(typeof ODD_PRESETS)[number]['id']>('equilibre');
  const [customOdds, setCustomOdds] = useState(false);
  const [minCombinedOdd, setMinCombinedOdd] = useState(10);
  const [maxCombinedOdd, setMaxCombinedOdd] = useState(25);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [minConfidence, setMinConfidence] = useState(0.55);
  const confidencePercent = clampConfidencePercent(Math.round(minConfidence * 100));
  const [coupon, setCoupon] = useState<{
    id: string;
    selections: CouponSelection[];
    disclaimer: string;
    combined_odd?: number;
    min_combined_odd?: number;
    max_combined_odd?: number;
    warning?: string;
  } | null>(null);

  useEffect(() => {
    if (!loading && !isAuthenticated) {
      navigate('/auth', { state: { from: '/coupon' } });
    }
  }, [isAuthenticated, loading, navigate]);

  useEffect(() => {
    if (customOdds) return;
    const selected = ODD_PRESETS.find((p) => p.id === preset) ?? ODD_PRESETS[1];
    setMinCombinedOdd(selected.min);
    setMaxCombinedOdd(selected.max);
  }, [preset, customOdds]);

  const generate = useMutation({
    mutationFn: () =>
      api.generateCoupon({
        minConfidence,
        maxSelections: 10,
        minCombinedOdd,
        maxCombinedOdd,
      }),
    onSuccess: (data) => {
      setCoupon({
        id: data.id,
        disclaimer: data.disclaimer ?? '',
        combined_odd: data.combined_odd,
        min_combined_odd: data.min_combined_odd,
        max_combined_odd: data.max_combined_odd,
        warning: data.warning,
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
          Choisissez votre profil de cote combinée, nous sélectionnons jusqu&apos;à 10 matchs adaptés.
        </p>
      </div>

      <div className="card mb-6 space-y-5">
        <div>
          <p className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-3">Profil de cote combinée</p>
          <div className="grid grid-cols-3 gap-2">
            {ODD_PRESETS.map((p) => (
              <button
                key={p.id}
                type="button"
                onClick={() => {
                  setPreset(p.id);
                  setCustomOdds(false);
                }}
                className={`rounded-xl border px-3 py-3 text-left transition-colors ${
                  !customOdds && preset === p.id
                    ? 'border-brand bg-brand/10 text-brand-dark dark:text-brand-light'
                    : 'border-slate-200 dark:border-navy-600 hover:border-brand/40'
                }`}
              >
                <p className="font-semibold text-sm">{p.label}</p>
                <p className="text-xs text-slate-500 mt-1">
                  {p.min} - {p.max}
                </p>
              </button>
            ))}
          </div>
        </div>

        <div>
          <button
            type="button"
            onClick={() => setCustomOdds((v) => !v)}
            className="text-sm font-medium text-brand-dark dark:text-brand-light hover:underline"
          >
            {customOdds ? 'Revenir aux profils' : 'Personnaliser l\'intervalle'}
          </button>
          {customOdds && (
            <div className="mt-3 grid grid-cols-2 gap-3">
              <label className="text-sm text-slate-600 dark:text-slate-400">
                Minimum
                <input
                  type="number"
                  min={5}
                  max={50}
                  step={1}
                  value={minCombinedOdd}
                  onChange={(e) => setMinCombinedOdd(clampOdd(parseInt(e.target.value, 10) || 5))}
                  className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 dark:border-navy-600 dark:bg-navy-900"
                />
              </label>
              <label className="text-sm text-slate-600 dark:text-slate-400">
                Maximum
                <input
                  type="number"
                  min={5}
                  max={50}
                  step={1}
                  value={maxCombinedOdd}
                  onChange={(e) => setMaxCombinedOdd(clampOdd(parseInt(e.target.value, 10) || 50))}
                  className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 dark:border-navy-600 dark:bg-navy-900"
                />
              </label>
            </div>
          )}
        </div>

        <div>
          <button
            type="button"
            onClick={() => setShowAdvanced((v) => !v)}
            className="text-sm text-slate-500 hover:text-slate-700 dark:hover:text-slate-300"
          >
            {showAdvanced ? 'Masquer les options avancées' : 'Options avancées'}
          </button>
          {showAdvanced && (
            <div className="mt-3">
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
            </div>
          )}
        </div>

        <button
          onClick={() => generate.mutate()}
          disabled={generate.isPending || minCombinedOdd > maxCombinedOdd}
          className="btn-primary w-full"
        >
          {generate.isPending ? 'Génération en cours...' : 'Générer le coupon'}
        </button>
        {generate.isPending && (
          <p className="text-xs text-slate-500 text-center">Analyse des matchs disponibles en cours...</p>
        )}
      </div>

      {errorMessage && (
        <div className="card mb-6 border-red-500/30 bg-red-500/5">
          <p className="text-red-300 text-sm">{errorMessage}</p>
        </div>
      )}

      {coupon && (
        <div className="card">
          <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
            <div>
              <h3 className="font-display text-lg font-semibold text-heading">Coupon généré</h3>
              {coupon.combined_odd != null && coupon.selections.length > 0 && (
                <p className="text-sm text-brand-dark dark:text-brand-light mt-1">
                  Cote combinée : {formatOdd(coupon.combined_odd)}
                  {(coupon.min_combined_odd != null && coupon.max_combined_odd != null) && (
                    <span className="text-slate-500">
                      {' '}
                      (objectif {coupon.min_combined_odd} - {coupon.max_combined_odd})
                    </span>
                  )}
                </p>
              )}
              {coupon.warning && (
                <div className="mt-2 rounded-lg border border-amber-500/40 bg-amber-500/10 px-3 py-2">
                  <p className="text-sm text-amber-800 dark:text-amber-200">{coupon.warning}</p>
                  <p className="text-xs text-amber-700/80 dark:text-amber-300/80 mt-1">
                    Pas assez de matchs à cotes élevées disponibles. Essayez le profil Prudent ou baissez la confiance minimum.
                  </p>
                </div>
              )}
            </div>
            {coupon.selections.length > 0 && (
              <button
                type="button"
                className="btn-secondary text-sm"
                onClick={() =>
                  exportCouponPdf({
                    id: coupon.id,
                    selections: coupon.selections,
                    combinedOdd: coupon.combined_odd,
                    disclaimer: coupon.disclaimer,
                    userName: user?.display_name,
                  })
                }
              >
                Exporter PDF
              </button>
            )}
          </div>
          {(coupon.selections?.length ?? 0) === 0 ? (
            <p className="text-slate-400">
              Aucune sélection ne correspond à vos critères pour le moment.
            </p>
          ) : (
            <ol className="space-y-3">
              {coupon.selections.map((sel, i) => {
                const category = sel.market_category ?? marketCategory(sel.market);
                const label = sel.market_label ?? marketLabel(sel.selection);
                const trend = oddTrendLabel(sel.odd_trend);
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
                      {sel.bookmaker_odd != null && sel.bookmaker_odd > 1 && (
                        <p className="text-xs text-slate-400 mt-1">
                          Meilleure cote : {formatOdd(sel.bookmaker_odd)}
                          {sel.bookmaker_name ? ` (${sel.bookmaker_name})` : ''}
                          {sel.avg_odd != null && sel.avg_odd > 1
                            ? ` · Moyenne : ${formatOdd(sel.avg_odd)}`
                            : ''}
                        </p>
                      )}
                      {trend && <p className="text-xs text-slate-500 mt-1">{trend}</p>}
                      {sel.value_edge != null && sel.value_edge > 0.05 && (
                        <span className="badge-high inline-block mt-2">Écart favorable</span>
                      )}
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
