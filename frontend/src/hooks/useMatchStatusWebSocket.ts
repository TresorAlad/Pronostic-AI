import { useEffect, useRef } from 'react';

interface StatusChangeEvent {
  match_id: string;
  from?: string;
  to?: string;
}

function wsURL() {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}/api/v1/ws/live`;
}

export function useMatchStatusWebSocket(onStatusChange?: (event: StatusChangeEvent) => void) {
  const callbackRef = useRef(onStatusChange);
  callbackRef.current = onStatusChange;

  useEffect(() => {
    if (!callbackRef.current) return;

    let ws: WebSocket | null = null;
    let retry: number | null = null;

    const connect = () => {
      try {
        ws = new WebSocket(wsURL());
        ws.onmessage = (event) => {
          try {
            const msg = JSON.parse(event.data);
            if (msg.type === 'match_status_change') {
              callbackRef.current?.(msg.data as StatusChangeEvent);
            }
          } catch {
            // ignore
          }
        };
        ws.onclose = () => {
          retry = window.setTimeout(connect, 5000);
        };
      } catch {
        retry = window.setTimeout(connect, 5000);
      }
    };

    connect();
    return () => {
      if (retry) window.clearTimeout(retry);
      ws?.close();
    };
  }, []);
}
