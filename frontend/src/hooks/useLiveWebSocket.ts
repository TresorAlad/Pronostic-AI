import { useEffect, useRef, useState, useCallback } from 'react';
import type { Match, Prediction } from '../api';

interface LiveEvent {
  type: string;
  data: unknown;
  ts: string;
}

function wsURL() {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}/api/v1/ws/live`;
}

export function useLiveWebSocket() {
  const [liveMatches, setLiveMatches] = useState<Match[]>([]);
  const [predictions, setPredictions] = useState<Record<string, Prediction>>({});
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

  return { liveMatches, predictions, connected };
}
