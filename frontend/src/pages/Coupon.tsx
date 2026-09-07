import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { api, confidenceBadge, formatProbability } from '../api';

interface CouponSelection {
  match_id: string;
  home_team: string;
  away_team: string;
  market: string;
  selection: string;
  confidence: number;
}

export default function Coupon() {
  const [minConfidence, setMinConfidence] = useState(0.65);
  const [coupon, setCoupon] = useState<{
    id: string;
    selections: CouponSelection[];
    disclaimer: string;
  } | null>(null);

  const generate = useMutation({
    mutationFn: () => api.generateCoupon(minConfidence, 5),
    onSuccess: (data) => setCoupon(data as typeof coupon),
  });

  return (
    <div className="max-w-2xl mx-auto">
      <h1 className="text-3xl font-bold mb-2">Coupon IA</h1>
      <p className="text-gray-400 mb-8">
        Selection automatique basee sur la confiance du modele ML
      </p>

      <div className="card mb-6">
        <label className="block text-sm text-gray-400 mb-2">
          Confiance minimum: {formatProbability(minConfidence)}
        </label>
        <input
          type="range"
          min={0.5}
          max={0.9}
          step={0.05}
          value={minConfidence}
          onChange={(e) => setMinConfidence(parseFloat(e.target.value))}
          className="w-full accent-accent"
        />
        <button
          onClick={() => generate.mutate()}
          disabled={generate.isPending}
          className="btn-primary w-full mt-4"
        >
          {generate.isPending ? 'Generation...' : 'Generer le coupon'}
        </button>
      </div>

      {coupon && (
        <div className="card">
          <h3 className="text-lg font-semibold mb-4">Coupon genere</h3>
          {coupon.selections.length === 0 ? (
            <p className="text-gray-400">Aucune selection ne depasse le seuil de confiance.</p>
          ) : (
            <ol className="space-y-4">
              {coupon.selections.map((sel, i) => (
                <li key={i} className="flex items-start gap-4 p-3 bg-pitch-900 rounded-lg">
                  <span className="text-accent font-bold text-lg">{i + 1}</span>
                  <div className="flex-1">
                    <p className="font-medium">
                      {sel.home_team} vs {sel.away_team}
                    </p>
                    <p className="text-sm text-gray-400 capitalize">
                      {sel.selection.replace(/_/g, ' ')}
                    </p>
                  </div>
                  <span className={confidenceBadge(sel.confidence)}>
                    {formatProbability(sel.confidence)}
                  </span>
                </li>
              ))}
            </ol>
          )}
          <p className="text-xs text-gray-500 mt-4 border-t border-pitch-700 pt-4">
            {coupon.disclaimer}
          </p>
        </div>
      )}
    </div>
  );
}
