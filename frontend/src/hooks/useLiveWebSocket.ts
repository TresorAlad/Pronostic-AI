import { useEffect, useRef, useState, useCallback } from 'react';
import type { Match, Prediction } from '../api';

interface LiveEvent {
  type: string;
  data: unknown;
  ts: string;
}

export function useLiveWebSocket() {
  const [liveMatches, setLiveMatches] = useState<Match[]>([]);
  const [predictions, setPredictions] = useState<Record<string, Prediction>>({});
  const [connected, setConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);

  const connect = useCallback(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.hostname;
    const ws = new WebSocket(`${protocol}//${host}:8080/api/v1/ws/live`);

    ws.onopen = () => setConnected(true);
    ws.onclose = () => {
      setConnected(false);
      setTimeout(connect, 5000);
    };

    ws.onmessage = (event) => {
      const msg: LiveEvent = JSON.parse(event.data);
      if (msg.type === 'match_update') {
        const data = msg.data as Match & { match_id?: string };
        setLiveMatches((prev) => {
          const id = data.match_id || data.id;
          const idx = prev.findIndex((m) => m.id === id);
          if (idx >= 0) {
            const updated = [...prev];
            updated[idx] = { ...updated[idx], ...data };
            return updated;
          }
          return [...prev, data as Match];
        });
      }
      if (msg.type === 'prediction_update') {
        const pred = msg.data as Prediction;
        setPredictions((prev) => ({ ...prev, [pred.match_id]: pred }));
      }
    };

    wsRef.current = ws;
  }, []);

  useEffect(() => {
    connect();
    return () => wsRef.current?.close();
  }, [connect]);

  return { liveMatches, predictions, connected };
}
