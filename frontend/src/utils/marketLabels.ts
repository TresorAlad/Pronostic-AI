export const MARKET_LABELS: Record<string, string> = {
  home_win: 'Victoire domicile',
  draw: 'Match nul',
  away_win: 'Victoire extérieure',
  over_1_5: 'Plus de 1,5 buts',
  over_2_5: 'Plus de 2,5 buts',
  over_3_5: 'Plus de 3,5 buts',
  under_1_5: 'Moins de 1,5 buts',
  under_2_5: 'Moins de 2,5 buts',
  btts: 'Les deux équipes marquent',
  btts_no: 'Les deux équipes ne marquent pas',
  team_over_0_5_home: 'Domicile marque au moins 1 but',
  team_over_1_5_home: 'Domicile marque plus de 1,5 buts',
  team_over_0_5_away: 'Extérieur marque au moins 1 but',
  team_over_1_5_away: 'Extérieur marque plus de 1,5 buts',
  result_btts_home_yes: 'Victoire domicile et les deux équipes marquent',
  result_btts_draw_yes: 'Match nul et les deux équipes marquent',
  result_btts_away_yes: 'Victoire extérieure et les deux équipes marquent',
  over_0_5_ht: 'Plus de 0,5 but en première mi-temps',
  draw_no_bet_home: 'Victoire domicile (remboursé si nul)',
  draw_no_bet_away: 'Victoire extérieure (remboursé si nul)',
  double_chance_1x: 'Double chance 1X (domicile ou nul)',
  double_chance_x2: 'Double chance X2 (nul ou extérieur)',
  double_chance_12: 'Double chance 12 (pas de nul)',
  over_corners_9_5: 'Plus de 9,5 corners',
  'over_corners_9.5': 'Plus de 9,5 corners',
  corner_winner_home: 'Domicile gagne aux corners',
  corner_winner_away: 'Extérieur gagne aux corners',
  over_shots_22_5: 'Plus de 22,5 tirs',
  over_shots_on_target_8_5: 'Plus de 8,5 tirs cadrés',
  over_fouls_20_5: 'Plus de 20,5 fautes',
  over_fouls_22_5: 'Plus de 22,5 fautes',
  over_fouls_25_5: 'Plus de 25,5 fautes',
  over_cards_3_5: 'Plus de 3,5 cartons',
  over_cards_4_5: 'Plus de 4,5 cartons',
  over_cards_5_5: 'Plus de 5,5 cartons',
  over_offsides_2_5: 'Plus de 2,5 hors-jeu',
  over_offsides_3_5: 'Plus de 3,5 hors-jeu',
  over_offsides_4_5: 'Plus de 4,5 hors-jeu',
  home_possession_over_50: 'Domicile > 50 % possession',
  away_possession_over_50: 'Extérieur > 50 % possession',
  predicted_total_corners: 'Total corners estimé',
  predicted_total_fouls: 'Total fautes estimé',
  predicted_total_cards: 'Total cartons estimé',
  predicted_total_offsides: 'Total hors-jeu estimé',
};

export const MARKET_CATEGORIES: { title: string; markets: string[] }[] = [
  {
    title: 'Temps réglementaire',
    markets: [
      'home_win',
      'draw',
      'away_win',
      'double_chance_1x',
      'double_chance_x2',
      'double_chance_12',
      'draw_no_bet_home',
      'draw_no_bet_away',
    ],
  },
  {
    title: 'Buts',
    markets: [
      'over_1_5',
      'over_2_5',
      'over_3_5',
      'under_1_5',
      'under_2_5',
      'btts',
      'btts_no',
      'team_over_0_5_home',
      'team_over_1_5_home',
      'team_over_0_5_away',
      'team_over_1_5_away',
      'result_btts_home_yes',
      'result_btts_draw_yes',
      'result_btts_away_yes',
      'over_0_5_ht',
    ],
  },
  {
    title: 'Tirs',
    markets: ['over_shots_22_5', 'over_shots_on_target_8_5'],
  },
  {
    title: 'Corners',
    markets: ['over_corners_9_5', 'over_corners_9.5', 'corner_winner_home', 'corner_winner_away', 'predicted_total_corners'],
  },
  {
    title: 'Cartons',
    markets: ['over_cards_3_5', 'over_cards_4_5', 'over_cards_5_5', 'predicted_total_cards'],
  },
  {
    title: 'Fautes',
    markets: ['over_fouls_20_5', 'over_fouls_22_5', 'over_fouls_25_5', 'predicted_total_fouls'],
  },
  {
    title: 'Hors-jeu',
    markets: ['over_offsides_2_5', 'over_offsides_3_5', 'over_offsides_4_5', 'predicted_total_offsides'],
  },
  {
    title: 'Possession',
    markets: ['home_possession_over_50', 'away_possession_over_50'],
  },
];

export function marketLabel(key: string): string {
  if (MARKET_LABELS[key]) return MARKET_LABELS[key];
  const normalized = key.replace('.', '_');
  if (MARKET_LABELS[normalized]) return MARKET_LABELS[normalized];
  return key.replace(/_/g, ' ');
}

const CATEGORY_BY_MARKET = new Map<string, string>(
  MARKET_CATEGORIES.flatMap((cat) => cat.markets.map((m) => [m, cat.title] as const))
);

export function marketCategory(key: string): string {
  const direct = CATEGORY_BY_MARKET.get(key) ?? CATEGORY_BY_MARKET.get(key.replace('.', '_'));
  if (direct) return direct;

  if (key.startsWith('double_chance_') || key.startsWith('draw_no_bet_')) return 'Temps réglementaire';
  if (key === 'home_win' || key === 'draw' || key === 'away_win') return 'Temps réglementaire';
  if (key.includes('corners') || key.startsWith('corner_winner_')) return 'Corners';
  if (key.includes('shots')) return 'Tirs';
  if (key.includes('cards')) return 'Cartons';
  if (key.includes('fouls')) return 'Fautes';
  if (key.includes('offsides')) return 'Hors-jeu';
  if (key.includes('possession')) return 'Possession';
  if (key.startsWith('team_over_') || key.startsWith('result_btts_') || key.startsWith('over_0_5_ht')) return 'Buts';
  if (key.startsWith('over_') || key.startsWith('under_') || key.startsWith('btts')) return 'Buts';
  return 'Autre';
}

const CATEGORY_COLORS: Record<string, string> = {
  'Temps réglementaire': 'bg-blue-500/10 text-blue-300 border-blue-500/30',
  Buts: 'bg-brand/10 text-brand-dark border-brand/30 dark:text-brand-light',
  Tirs: 'bg-purple-500/10 text-purple-300 border-purple-500/30',
  Corners: 'bg-cyan-500/10 text-cyan-300 border-cyan-500/30',
  Cartons: 'bg-gold/10 text-gold-light border-gold/30',
  Fautes: 'bg-red-500/10 text-red-300 border-red-500/30',
  'Hors-jeu': 'bg-pink-500/10 text-pink-300 border-pink-500/30',
  Possession: 'bg-indigo-500/10 text-indigo-300 border-indigo-500/30',
  Autre: 'bg-slate-100 text-slate-700 border-slate-200 dark:bg-navy-700 dark:text-slate-300 dark:border-navy-600',
};

export function marketCategoryClass(category: string): string {
  return CATEGORY_COLORS[category] ?? CATEGORY_COLORS.Autre;
}

export function groupPredictions(predictions: Record<string, number> | null | undefined) {
  if (!predictions) return [];
  const used = new Set<string>();
  const groups = MARKET_CATEGORIES.map((cat) => {
    const items = cat.markets
      .filter((m) => predictions[m] !== undefined)
      .map((m) => {
        used.add(m);
        return { key: m, value: predictions[m] };
      });
    return { title: cat.title, items };
  }).filter((g) => g.items.length > 0);

  const other = Object.entries(predictions)
    .filter(([k]) => !used.has(k) && !k.startsWith('double_chance_') && !k.startsWith('predicted_'))
    .map(([key, value]) => ({ key, value }));

  if (other.length > 0) {
    groups.push({ title: 'Autres', items: other });
  }

  return groups;
}
