const API_URL = import.meta.env.VITE_API_URL || '/api/v1';

export interface Team {
  id: string;
  name: string;
  logo_url?: string;
}

export interface Match {
  id: string;
  external_id: number;
  league_name: string;
  home_team: Team;
  away_team: Team;
  kickoff_at: string;
  status: string;
  minute?: number;
  home_score?: number;
  away_score?: number;
  venue?: string;
  round?: string;
}

export interface Prediction {
  id: string;
  match_id: string;
  model_version: string;
  predictions: Record<string, number>;
  confidence: Record<string, number>;
  no_bet_recommended: boolean;
  ai_analysis?: string;
  ai_reasons?: string[];
  ai_abstain?: boolean;
  is_live?: boolean;
  delta?: Record<string, number>;
  direction?: Record<string, 'up' | 'down' | 'flat'>;
  minute?: number;
}

export interface CouponSelection {
  match_id: string;
  home_team: string;
  away_team: string;
  league_name?: string;
  market: string;
  selection: string;
  market_category?: string;
  market_label?: string;
  confidence: number;
  value_edge?: number;
  bookmaker_odd?: number;
  avg_odd?: number;
  bookmaker_name?: string;
  odd_trend?: 'up' | 'down' | 'flat';
}

export interface GenerateCouponOptions {
  minConfidence?: number;
  maxSelections?: number;
  minCombinedOdd?: number;
  maxCombinedOdd?: number;
  name?: string;
}

export interface ModelPerformance {
  model_name: string;
  model_version: string;
  market: string;
  accuracy?: number;
  log_loss?: number;
  brier_score?: number;
  sample_size?: number;
}

export interface PerformanceTrend {
  period: string;
  market: string;
  accuracy: number;
  sample_size: number;
}

export interface SavedCouponSummary {
  id: string;
  name: string;
  created_at: string;
  selection_count: number;
}

export interface User {
  id: string;
  email: string;
  display_name: string;
  created_at?: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export interface MeResponse {
  user: User;
  coupon_count: number;
}

export interface PublicStats {
  matches_today: number;
  live_matches: number;
  finished_matches: number;
  match_statistics: number;
  predictions: number;
  outcomes: number;
}

export interface League {
  id: string;
  external_id: number;
  name: string;
  country?: string;
  logo_url?: string;
}

export interface LeagueFilterOption {
  id: string;
  external_id: number;
  label: string;
  match_count: number;
}

export interface TrackedLeague {
  external_id: number;
  label: string;
  priority: number;
}

export interface MatchDayQuery {
  date?: string;
  leagueId?: string;
  leagueExternalId?: number;
  status?: 'scheduled' | 'live' | 'finished' | 'all';
}

function buildQuery(params: Record<string, string | undefined>): string {
  const q = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value && value !== 'all') q.set(key, value);
  }
  const s = q.toString();
  return s ? `?${s}` : '';
}

export interface AppNotification {
  id: string;
  type: string;
  title: string;
  body: string;
  read_at?: string;
  created_at: string;
}

export interface MatchOddRow {
  bookmaker: string;
  market: string;
  selection: string;
  odd: number;
  implied_probability?: number;
  ml_probability?: number;
  value_edge?: number;
}

export interface UserPerformanceData {
  summary: {
    total_selections: number;
    correct_count: number;
    accuracy: number;
    by_market: Record<string, number>;
  };
  trend: Array<{ period: string; accuracy: number; sample_size: number }>;
  recent: Array<Record<string, unknown>>;
}

function parseRecord(value: unknown): Record<string, number> {
  if (!value) return {};
  if (typeof value === 'string') {
    try {
      return JSON.parse(value) as Record<string, number>;
    } catch {
      return {};
    }
  }
  if (typeof value === 'object') {
    return value as Record<string, number>;
  }
  return {};
}

export function normalizePrediction(raw: Record<string, unknown>): Prediction {
  const directionRaw = raw.direction;
  let direction: Prediction['direction'];
  if (directionRaw && typeof directionRaw === 'object') {
    direction = directionRaw as Prediction['direction'];
  }
  return {
    ...(raw as unknown as Prediction),
    predictions: parseRecord(raw.predictions),
    confidence: parseRecord(raw.confidence),
    delta: parseRecord(raw.delta),
    direction,
    ai_reasons: Array.isArray(raw.ai_reasons) ? (raw.ai_reasons as string[]) : undefined,
  };
}

async function fetchAPI<T>(path: string, options?: RequestInit): Promise<T> {
  const token = localStorage.getItem('token');
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  };
  const res = await fetch(`${API_URL}${path}`, { ...options, headers });
  if (!res.ok) {
    const body = await res.text();
    let message = `Erreur API (${res.status})`;
    try {
      const parsed = JSON.parse(body);
      if (parsed.error) message = parsed.error;
    } catch {
      if (body) message = body;
    }
    throw new Error(message);
  }
  return res.json();
}

export const api = {
  getMatchesToday: (query: MatchDayQuery = {}) => {
    const status = query.status === 'all' ? undefined : query.status;
    return fetchAPI<Match[]>(
      `/matches/today${buildQuery({
        date: query.date,
        league_id: query.leagueId,
        league_external_id:
          query.leagueExternalId != null ? String(query.leagueExternalId) : undefined,
        status,
      })}`
    );
  },
  getActiveLeagues: (date: string) =>
    fetchAPI<LeagueFilterOption[]>(`/leagues/active${buildQuery({ date })}`),
  getTrackedLeagues: () => fetchAPI<TrackedLeague[]>('/leagues/tracked'),
  getLiveMatches: () => fetchAPI<Match[]>('/matches/live'),
  getMatch: (id: string) => fetchAPI<Match>(`/matches/${id}`),
  getMatchStats: (id: string) => fetchAPI<unknown[]>(`/matches/${id}/stats`),
  getPrediction: async (id: string) =>
    normalizePrediction((await fetchAPI<Record<string, unknown>>(`/matches/${id}/prediction`)) as Record<string, unknown>),
  analyzeMatch: async (id: string) =>
    normalizePrediction(
      (await fetchAPI<Record<string, unknown>>(`/matches/${id}/analyze`, { method: 'POST' })) as Record<string, unknown>
    ),
  getPerformance: () => fetchAPI<ModelPerformance[]>('/predictions/performance'),
  getPerformanceTrend: () => fetchAPI<PerformanceTrend[]>('/predictions/performance/trend'),
  runEvaluation: () =>
    fetchAPI<{ evaluated: number; message: string }>('/evaluation/run', { method: 'POST' }),
  getMyCoupons: () => fetchAPI<SavedCouponSummary[]>('/coupons/mine'),
  getMyCoupon: (id: string) => fetchAPI<Record<string, unknown>>(`/coupons/mine/${id}`),
  generateCoupon: (options: GenerateCouponOptions = {}) => {
    const {
      minConfidence = 0.55,
      maxSelections = 10,
      minCombinedOdd = 10,
      maxCombinedOdd = 25,
      name,
    } = options;
    return fetchAPI<{
      id: string;
      selections: CouponSelection[] | null;
      disclaimer: string;
      combined_odd?: number;
      min_combined_odd?: number;
      max_combined_odd?: number;
      warning?: string;
    }>('/coupons/generate', {
      method: 'POST',
      body: JSON.stringify({
        min_confidence: minConfidence,
        max_selections: maxSelections,
        min_combined_odd: minCombinedOdd,
        max_combined_odd: maxCombinedOdd,
        name,
      }),
    });
  },
  login: (email: string, password: string) =>
    fetchAPI<AuthResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    }),
  register: (email: string, password: string, displayName: string) =>
    fetchAPI<AuthResponse>('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, password, display_name: displayName }),
    }),
  getMe: () => fetchAPI<MeResponse>('/auth/me'),
  updateMe: (displayName: string) =>
    fetchAPI<MeResponse>('/auth/me', {
      method: 'PATCH',
      body: JSON.stringify({ display_name: displayName }),
    }),
  getPublicStats: () => fetchAPI<PublicStats>('/stats/public'),
  prewarmPredictions: () =>
    fetchAPI<{ warmed: number }>('/predictions/prewarm', { method: 'POST' }),
  getNotifications: () => fetchAPI<AppNotification[]>('/notifications'),
  markNotificationRead: (id: string) =>
    fetchAPI<void>(`/notifications/${id}/read`, { method: 'PATCH' }),
  getMyPerformance: () => fetchAPI<UserPerformanceData>('/performance/mine'),
  getMatchOdds: (id: string) => fetchAPI<MatchOddRow[]>(`/matches/${id}/odds`),
};

export function confidenceBadge(confidence: number) {
  if (confidence >= 0.7) return 'badge-high';
  if (confidence >= 0.55) return 'badge-medium';
  return 'badge-low';
}

export function formatProbability(p: number) {
  return `${Math.round(p * 100)}%`;
}

export function formatOdd(n: number) {
  return n.toFixed(2).replace('.', ',');
}

export function isLoggedIn() {
  return Boolean(localStorage.getItem('token') && localStorage.getItem('user'));
}
