import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { useAuth, userLabel } from '../hooks/useAuth';
import { api } from '../api';

export default function Account() {
  const { user, couponCount, loading, isAuthenticated, refresh } = useAuth();
  const [displayName, setDisplayName] = useState('');
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState('');

  const { data: coupons } = useQuery({
    queryKey: ['my-coupons'],
    queryFn: api.getMyCoupons,
    enabled: isAuthenticated,
  });

  if (loading) {
    return <p className="text-slate-400">Chargement du compte...</p>;
  }

  if (!isAuthenticated || !user) {
    return (
      <div className="card mx-auto max-w-lg text-center py-12">
        <p className="text-slate-400">Vous n&apos;êtes pas connecté.</p>
        <Link to="/auth" className="btn-primary mt-4 inline-block">
          Se connecter
        </Link>
      </div>
    );
  }

  const memberSince = user.created_at
    ? new Date(user.created_at).toLocaleDateString('fr-FR', {
        day: 'numeric',
        month: 'long',
        year: 'numeric',
      })
    : null;

  const recentCoupons = coupons?.slice(0, 3) ?? [];

  const handleSaveDisplayName = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!displayName.trim()) return;
    setSaving(true);
    setSaveError('');
    try {
      await api.updateMe(displayName.trim());
      await refresh();
    } catch {
      setSaveError('Impossible de mettre à jour le profil.');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="mx-auto max-w-3xl">
      <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <p className="text-sm font-semibold uppercase tracking-wider text-brand-dark dark:text-brand-light">
            Mon compte
          </p>
          <h1 className="page-title mt-1">Bonjour, {userLabel(user)}</h1>
          <p className="page-subtitle">Aperçu de votre profil et de votre activité sur Pronostic AI.</p>
        </div>
        <button type="button" onClick={() => refresh()} className="btn-secondary text-sm shrink-0">
          Actualiser
        </button>
      </div>

      <div className="card mb-6 flex flex-col gap-4 sm:flex-row sm:items-center">
        <span className="user-avatar user-avatar-xl shrink-0">
          {(user.display_name || user.email).slice(0, 2).toUpperCase()}
        </span>
        <div className="flex-1 min-w-0">
          <h2 className="font-display text-xl font-semibold text-heading">{userLabel(user)}</h2>
          <p className="text-slate-500 dark:text-slate-400">{user.email}</p>
          <div className="mt-3 flex flex-wrap gap-2">
            <span className="user-status-badge">Session active</span>
            {memberSince && (
              <span className="rounded-full border border-slate-200 px-2.5 py-0.5 text-xs text-slate-500 dark:border-navy-600">
                Membre depuis {memberSince}
              </span>
            )}
          </div>
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-3 mb-6">
        <StatCard label="Coupons sauvegardés" value={String(couponCount)} />
        <StatCard label="Derniers coupons" value={String(recentCoupons.length)} hint="affichés ci-dessous" />
        <StatCard label="Performance" value="Voir stats" hint="sélections évaluées" link="/my-performance" />
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        <div className="card">
          <h3 className="font-display text-lg font-semibold text-heading mb-4">Profil</h3>
          <form onSubmit={handleSaveDisplayName} className="space-y-3">
            <input
              type="text"
              placeholder={user.display_name || 'Nom d\'affichage'}
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              className="input-field"
            />
            {saveError && <p className="text-red-400 text-sm">{saveError}</p>}
            <button type="submit" disabled={saving || !displayName.trim()} className="btn-secondary w-full text-sm">
              {saving ? 'Enregistrement...' : 'Mettre à jour le nom'}
            </button>
          </form>
        </div>

        <div className="card">
          <h3 className="font-display text-lg font-semibold text-heading mb-4">Actions rapides</h3>
          <div className="flex flex-col gap-2">
            <Link to="/coupon" className="btn-primary text-center">
              Générer un coupon
            </Link>
            <Link to="/my-coupons" className="btn-secondary text-center">
              Voir tous mes coupons
            </Link>
            <Link to="/my-performance" className="btn-secondary text-center">
              Ma performance
            </Link>
            <Link to="/dashboard" className="btn-secondary text-center">
              Matchs du jour
            </Link>
          </div>
        </div>
      </div>

      <div className="card mt-6">
          <h3 className="font-display text-lg font-semibold text-heading mb-4">Derniers coupons</h3>
          {recentCoupons.length === 0 ? (
            <p className="text-sm text-slate-500">Aucun coupon pour le moment.</p>
          ) : (
            <ul className="space-y-3">
              {recentCoupons.map((c) => (
                <li
                  key={c.id}
                  className="flex items-center justify-between gap-3 rounded-lg border border-slate-200 px-3 py-2 dark:border-navy-600"
                >
                  <div className="min-w-0">
                    <p className="font-medium text-heading truncate">{c.name}</p>
                    <p className="text-xs text-slate-500">
                      {new Date(c.created_at).toLocaleDateString('fr-FR')} · {c.selection_count} sélections
                    </p>
                  </div>
                  <Link to="/my-coupons" className="text-xs font-medium text-brand-dark dark:text-brand-light shrink-0">
                    Voir
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </div>
    </div>
  );
}

function StatCard({ label, value, hint, link }: { label: string; value: string; hint?: string; link?: string }) {
  const inner = (
    <>
      <p className="text-2xl font-display font-bold text-brand-dark dark:text-brand-light">{value}</p>
      <p className="mt-1 text-sm font-medium text-heading">{label}</p>
      {hint && <p className="mt-1 text-xs text-slate-500">{hint}</p>}
    </>
  );
  if (link) {
    return (
      <Link to={link} className="card text-center block hover:border-brand/30 transition-colors">
        {inner}
      </Link>
    );
  }
  return <div className="card text-center">{inner}</div>;
}
