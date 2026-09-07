import { useEffect, useRef, useState, useCallback } from 'react';
import type { Match, Prediction } from '../api';
import { normalizePrediction } from '../api';

const HISTORY_LEN = 20;

interface LiveEvent {
  type: string;
  data: unknown;
  ts: string;
}

export type ProbHistory = Record<string, Record<string, number[]>>;

function wsURL() {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}/api/v1/ws/live`;
}

function isTeamObject(value: unknown): value is Match['home_team'] {
  return typeof value === 'object' && value !== null && 'name' in value;
}

function mergeMatchUpdate(existing: Match | undefined, data: Record<string, unknown>): Match {
  const base = existing ?? ({} as Match);
  const homeTeam = isTeamObject(data.home_team)
    ? data.home_team
    : typeof data.home_team === 'string' && base.home_team
      ? { ...base.home_team, name: data.home_team }
      : base.home_team;
  const awayTeam = isTeamObject(data.away_team)
    ? data.away_team
    : typeof data.away_team === 'string' && base.away_team
      ? { ...base.away_team, name: data.away_team }
      : base.away_team;

  return {
    ...base,
    ...data,
    id: String(data.id ?? data.match_id ?? base.id ?? ''),
    home_team: homeTeam ?? { name: 'Domicile' },
    away_team: awayTeam ?? { name: 'Extérieur' },
    league_name: String(data.league_name ?? base.league_name ?? ''),
    status: String(data.status ?? base.status ?? 'live'),
  } as Match;
}

function appendHistory(
  prev: ProbHistory,
  matchId: string,
  predictions: Record<string, number>
): ProbHistory {
  const next = { ...prev, [matchId]: { ...(prev[matchId] ?? {}) } };
  for (const [market, value] of Object.entries(predictions)) {
    const series = [...(next[matchId][market] ?? []), value];
    next[matchId][market] = series.slice(-HISTORY_LEN);
  }
  return next;
}

export function useLiveWebSocket() {
  const [liveMatches, setLiveMatches] = useState<Match[]>([]);
  const [predictions, setPredictions] = useState<Record<string, Prediction>>({});
  const [probHistory, setProbHistory] = useState<ProbHistory>({});
  const [connected, setConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const retryRef = useRef<number | null>(null);

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) return;

    try {
      const ws = new WebSocket(wsURL());

      ws.onopen = () => setConnected(true);
      ws.onclose = () => {
        setConnected(false);
        retryRef.current = window.setTimeout(connect, 5000);
      };
      ws.onerror = () => ws.close();

      ws.onmessage = (event) => {
        try {
          const msg: LiveEvent = JSON.parse(event.data);
          if (msg.type === 'match_update') {
            const data = msg.data as Record<string, unknown>;
            const id = String(data.match_id ?? data.id ?? '');
            if (!id) return;

            setLiveMatches((prev) => {
              const idx = prev.findIndex((m) => m.id === id);
              if (idx >= 0) {
                const updated = [...prev];
                updated[idx] = mergeMatchUpdate(updated[idx], data);
                return updated;
              }
              return [...prev, mergeMatchUpdate(undefined, data)];
            });
          }
          if (msg.type === 'prediction_update') {
            const pred = normalizePrediction(msg.data as Record<string, unknown>);
            if (pred.match_id) {
              setPredictions((prev) => ({ ...prev, [pred.match_id]: pred }));
              if (pred.predictions) {
                setProbHistory((prev) => appendHistory(prev, pred.match_id, pred.predictions));
              }
            }
          }
        } catch {
          // ignore malformed websocket payloads
        }
      };

      wsRef.current = ws;
    } catch {
      setConnected(false);
      retryRef.current = window.setTimeout(connect, 5000);
    }
  }, []);

  useEffect(() => {
    connect();
    return () => {
      if (retryRef.current) window.clearTimeout(retryRef.current);
      wsRef.current?.close();
    };
  }, [connect]);

  return { liveMatches, predictions, probHistory, connected };
}
