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

export interface ModelPerformance {
  model_name: string;
  model_version: string;
  market: string;
  accuracy?: number;
  log_loss?: number;
  brier_score?: number;
  sample_size?: number;
}

async function fetchAPI<T>(path: string, options?: RequestInit): Promise<T> {
  const token = localStorage.getItem('token');
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  };
  const res = await fetch(`${API_URL}${path}`, { ...options, headers });
  if (!res.ok) throw new Error(`API error: ${res.status}`);
  return res.json();
}

export const api = {
  getMatchesToday: () => fetchAPI<Match[]>('/matches/today'),
  getLiveMatches: () => fetchAPI<Match[]>('/matches/live'),
  getMatch: (id: string) => fetchAPI<Match>(`/matches/${id}`),
  getMatchStats: (id: string) => fetchAPI<unknown[]>(`/matches/${id}/stats`),
  getPrediction: (id: string) => fetchAPI<Prediction>(`/matches/${id}/prediction`),
  analyzeMatch: (id: string) => fetchAPI<Prediction>(`/matches/${id}/analyze`, { method: 'POST' }),
  getPerformance: () => fetchAPI<ModelPerformance[]>('/predictions/performance'),
  generateCoupon: (minConfidence = 0.65, maxSelections = 5) =>
    fetchAPI<{ id: string; selections: unknown[]; disclaimer: string }>('/coupons/generate', {
      method: 'POST',
      body: JSON.stringify({ min_confidence: minConfidence, max_selections: maxSelections }),
    }),
  login: (email: string, password: string) =>
    fetchAPI<{ token: string }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    }),
  register: (email: string, password: string, displayName: string) =>
    fetchAPI<{ token: string }>('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, password, display_name: displayName }),
    }),
};

export function confidenceBadge(confidence: number) {
  if (confidence >= 0.7) return 'badge-high';
  if (confidence >= 0.55) return 'badge-medium';
  return 'badge-low';
}

export function formatProbability(p: number) {
  return `${Math.round(p * 100)}%`;
}
