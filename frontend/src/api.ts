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
}

export interface CouponSelection {
  match_id: string;
  home_team: string;
  away_team: string;
  market: string;
  selection: string;
  market_category?: string;
  market_label?: string;
  confidence: number;
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
  return {
    ...(raw as unknown as Prediction),
    predictions: parseRecord(raw.predictions),
    confidence: parseRecord(raw.confidence),
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
  getMatchesToday: () => fetchAPI<Match[]>('/matches/today'),
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
  generateCoupon: (minConfidence = 0.55, maxSelections = 8) =>
    fetchAPI<{ id: string; selections: CouponSelection[] | null; disclaimer: string }>(
      '/coupons/generate',
      {
        method: 'POST',
        body: JSON.stringify({ min_confidence: minConfidence, max_selections: maxSelections }),
      }
    ),
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
};

export function confidenceBadge(confidence: number) {
  if (confidence >= 0.7) return 'badge-high';
  if (confidence >= 0.55) return 'badge-medium';
  return 'badge-low';
}

export function formatProbability(p: number) {
  return `${Math.round(p * 100)}%`;
}

export function isLoggedIn() {
  return Boolean(localStorage.getItem('token') && localStorage.getItem('user'));
}
