import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { useAuth, userLabel } from '../hooks/useAuth';
import { api } from '../api';
import LandingLivePreview from '../components/LandingLivePreview';

const STEPS = [
  {
    step: '01',
    title: 'Collecte des données',
    text: 'Historique des matchs, statistiques détaillées et infos équipes mises à jour régulièrement.',
  },
  {
    step: '02',
    title: 'Estimations par marché',
    text: 'Des probabilités calibrées pour chaque type de pari, avec un niveau de confiance associé.',
  },
  {
    step: '03',
    title: 'Coupon et suivi',
    text: 'Générez un coupon diversifié, sauvegardez-le et consultez votre historique personnel.',
  },
];

const FEATURES = [
  {
    title: 'Probabilités fiables',
    description:
      'Estimations issues de l\'analyse de milliers de matchs terminés, pas de chiffres inventés.',
  },
  {
    title: 'Analyse IA explicable',
    description:
      "Un commentaire clair sur chaque rencontre, avec abstention si la confiance est insuffisante.",
  },
  {
    title: 'Coupons intelligents',
    description:
      'Sélections variées par catégorie de marché, avec niveau de confiance ajustable et mémorisation en compte.',
  },
  {
    title: 'Performance traçable',
    description:
      'Comparaison aux résultats réels pour suivre la qualité des prédictions dans le temps.',
  },
];

const MARKETS = [
  '1X2',
  'Over / Under',
  'Les deux marquent',
  'Double chance',
  'Corners',
  'Tirs',
  'Cartons',
  'Fautes',
  'Hors-jeu',
  'Possession',
];

function formatCount(n: number | undefined | null, loading = false) {
  if (loading) return '...';
  if (n == null) return '-';
  if (n >= 1000) return `${Math.round(n / 100) / 10}k`;
  return String(n);
}

export default function Landing() {
  const { isAuthenticated, user, couponCount } = useAuth();
  const { data: trackedLeagues } = useQuery({
    queryKey: ['tracked-leagues'],
    queryFn: api.getTrackedLeagues,
    staleTime: 300_000,
  });
  const { data: stats, isLoading, isError } = useQuery({
    queryKey: ['public-stats'],
    queryFn: api.getPublicStats,
    staleTime: 60_000,
    refetchInterval: 60_000,
    retry: 2,
  });

  const statsReady = !isLoading && !isError && stats != null;

  const liveStats = [
    {
      value: statsReady ? formatCount(stats.live_matches) : formatCount(undefined, isLoading),
      label: 'Matchs en direct',
      sub: statsReady
        ? `${stats.matches_today} match${stats.matches_today > 1 ? 's' : ''} aujourd'hui`
        : isError
          ? 'Données indisponibles'
          : 'Mise à jour en direct',
    },
    {
      value: formatCount(stats?.finished_matches, isLoading),
      label: 'Matchs analysés',
      sub: 'Grands championnats européens',
    },
    {
      value: formatCount(stats?.match_statistics, isLoading),
      label: 'Stats détaillées',
      sub: 'Corners, tirs, cartons',
    },
    {
      value: isAuthenticated ? String(couponCount) : formatCount(stats?.predictions, isLoading),
      label: isAuthenticated ? 'Vos coupons' : 'Prédictions',
      sub: isAuthenticated ? 'Sauvegardés sur votre compte' : 'Tous les marchés',
    },
  ];

  return (
    <div className="overflow-hidden">
      <section className="relative px-4 pb-16 pt-10 md:pb-24 md:pt-14">
        <div className="pointer-events-none absolute inset-0 -z-10">
          <div className="absolute left-1/2 top-0 h-[480px] w-[800px] -translate-x-1/2 rounded-full bg-emerald-500/10 blur-3xl dark:bg-emerald-500/15" />
          <div className="absolute -left-20 top-40 h-72 w-72 rounded-full bg-amber-500/10 blur-3xl" />
        </div>

        <div className="mx-auto grid max-w-6xl items-center gap-12 lg:grid-cols-2 lg:gap-16">
          <div className="text-center lg:text-left">
            {isAuthenticated && user && (
              <p className="mb-4 inline-flex items-center gap-2 rounded-full border border-emerald-500/25 bg-emerald-500/10 px-4 py-1.5 text-sm text-brand-dark dark:text-brand-light">
                <span className="h-2 w-2 rounded-full bg-emerald-500" />
                Connecté en tant que {userLabel(user)}
              </p>
            )}
            <p className="mb-3 text-sm font-semibold uppercase tracking-widest text-brand-dark dark:text-brand-light">
              Football Intelligence
            </p>
            <h1 className="font-display text-4xl font-bold leading-tight text-heading md:text-5xl">
              Pronostics football
              <span className="block text-brand-dark dark:text-brand-light">pilotés par la data.</span>
            </h1>
            <p className="mt-5 text-lg leading-relaxed text-slate-600 dark:text-slate-300">
              Statistiques match par match et analyse IA sur les cinq grands championnats
              européens. Transparent, mesurable, sans promesses impossibles.
            </p>
            <div className="mt-8 flex flex-wrap justify-center gap-3 lg:justify-start">
              <Link to="/dashboard" className="btn-primary px-8 py-3 text-base">
                Explorer les matchs
              </Link>
              {isAuthenticated ? (
                <Link to="/account" className="btn-secondary px-8 py-3 text-base">
                  Mon compte
                </Link>
              ) : (
                <Link to="/auth" className="btn-secondary px-8 py-3 text-base">
                  Créer un compte
                </Link>
              )}
            </div>
          </div>

          <LandingLivePreview />
        </div>
      </section>

      <section className="border-y border-slate-200/80 bg-white/60 py-10 dark:border-navy-600/50 dark:bg-navy-900/50">
        <div className="mx-auto grid max-w-5xl grid-cols-2 gap-6 px-4 md:grid-cols-4">
          {liveStats.map((s) => (
            <div key={s.label} className="text-center">
              <p className="font-display text-3xl font-bold text-brand-dark dark:text-brand-light">{s.value}</p>
              <p className="mt-1 text-sm font-semibold text-heading">{s.label}</p>
              <p className="text-xs text-slate-500">{s.sub}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="px-4 py-10">
        <div className="mx-auto max-w-5xl text-center">
          <p className="mb-4 text-xs font-semibold uppercase tracking-wider text-slate-500">
            Championnats couverts
          </p>
          <div className="flex flex-wrap items-center justify-center gap-3">
            {trackedLeagues?.map((league) => (
              <span key={league.external_id} className="landing-league-pill">
                {league.label}
              </span>
            )) ?? null}
          </div>
        </div>
      </section>

      <section className="px-4 py-16 md:py-20 bg-gradient-to-b from-transparent to-emerald-500/[0.03]">
        <div className="mx-auto max-w-5xl">
          <h2 className="font-display text-3xl font-bold text-center text-heading md:text-4xl mb-3">
            Comment ça marche
          </h2>
          <p className="text-center text-slate-600 dark:text-slate-400 mb-12 max-w-2xl mx-auto">
            Une chaîne complète, de la donnée brute à la recommandation affichée dans l&apos;interface.
          </p>
          <div className="grid gap-6 md:grid-cols-3">
            {STEPS.map((step) => (
              <div key={step.step} className="card relative pt-8">
                <span className="landing-step-num">{step.step}</span>
                <h3 className="font-display text-lg font-semibold text-heading">{step.title}</h3>
                <p className="mt-2 text-sm leading-relaxed text-slate-600 dark:text-slate-400">{step.text}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="px-4 py-16 md:py-20">
        <div className="mx-auto max-w-5xl">
          <h2 className="font-display text-3xl font-bold text-center text-heading md:text-4xl mb-12">
            Pourquoi Pronostic AI
          </h2>
          <div className="grid gap-5 sm:grid-cols-2">
            {FEATURES.map((f) => (
              <div key={f.title} className="card landing-feature-card">
                <h3 className="font-display text-lg font-semibold text-heading">{f.title}</h3>
                <p className="mt-2 text-sm leading-relaxed text-slate-600 dark:text-slate-400">{f.description}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="px-4 pb-16 md:pb-24">
        <div className="mx-auto max-w-5xl">
          <div className="card landing-markets-card p-8 md:p-12">
            <div className="grid gap-10 md:grid-cols-2 md:items-center">
              <div>
                <h2 className="font-display text-2xl font-bold text-heading md:text-3xl">
                  Des marchés variés, une seule source de vérité
                </h2>
                <p className="mt-4 text-slate-600 dark:text-slate-400">
                  Chaque probabilité affichée repose sur des statistiques réelles et des
                  estimations vérifiées. L&apos;IA ne crée jamais de chiffres de toutes pièces.
                </p>
                <Link to="/performance" className="btn-secondary mt-6 inline-block text-sm">
                  Consulter la performance
                </Link>
              </div>
              <div className="flex flex-wrap gap-2">
                {MARKETS.map((m) => (
                  <span key={m} className="landing-market-tag">
                    {m}
                  </span>
                ))}
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="border-t border-slate-200/80 px-4 py-16 dark:border-navy-600/50">
        <div className="mx-auto max-w-2xl text-center">
          <h2 className="font-display text-2xl font-bold text-heading md:text-3xl">
            {isAuthenticated ? 'Continuez votre analyse' : 'Rejoignez Pronostic AI'}
          </h2>
          <p className="mt-3 text-slate-600 dark:text-slate-400">
            {isAuthenticated
              ? 'Retrouvez vos coupons, générez de nouvelles sélections et suivez le live.'
              : 'Compte gratuit pour sauvegarder vos coupons et retrouver votre historique.'}
          </p>
          <div className="mt-8 flex flex-wrap justify-center gap-4">
            {isAuthenticated ? (
              <>
                <Link to="/coupon" className="btn-primary px-8">
                  Générer un coupon
                </Link>
                <Link to="/live" className="btn-secondary px-8">
                  Matchs live
                </Link>
              </>
            ) : (
              <>
                <Link to="/auth" className="btn-primary px-8">
                  Créer un compte
                </Link>
                <Link to="/dashboard" className="btn-secondary px-8">
                  Voir les matchs
                </Link>
              </>
            )}
          </div>
        </div>
      </section>
    </div>
  );
}
