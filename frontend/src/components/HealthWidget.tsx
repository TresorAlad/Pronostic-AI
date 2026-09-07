import { useEffect, useState } from 'react';

const API_URL = import.meta.env.VITE_API_URL || '/api/v1';

function backendHealthUrl() {
  const base = import.meta.env.VITE_BACKEND_URL;
  if (base) return `${base.replace(/\/$/, '')}/health`;
  if (API_URL.startsWith('http')) {
    return `${API_URL.replace(/\/api\/v1\/?$/, '')}/health`;
  }
  return '/health';
}
const ML_URL = import.meta.env.VITE_ML_URL || 'http://localhost:5002';
const AI_URL = import.meta.env.VITE_AI_URL || 'http://localhost:5001';

type ServiceStatus = 'ok' | 'down' | 'loading';

async function ping(url: string): Promise<boolean> {
  try {
    const res = await fetch(url, { signal: AbortSignal.timeout(3000) });
    return res.ok;
  } catch {
    return false;
  }
}

async function pingAI(): Promise<{ ok: boolean; neo4j?: boolean }> {
  try {
    const res = await fetch(`${AI_URL}/health`, { signal: AbortSignal.timeout(3000) });
    if (!res.ok) return { ok: false };
    const data = (await res.json()) as { neo4j?: boolean };
    return { ok: true, neo4j: data.neo4j };
  } catch {
    return { ok: false };
  }
}

export default function HealthWidget() {
  const [backend, setBackend] = useState<ServiceStatus>('loading');
  const [ml, setMl] = useState<ServiceStatus>('loading');
  const [ai, setAi] = useState<ServiceStatus>('loading');
  const [neo4j, setNeo4j] = useState<ServiceStatus>('loading');

  useEffect(() => {
    let cancelled = false;
    const check = async () => {
      const [be, mlOk, aiResult] = await Promise.all([
        ping(backendHealthUrl()),
        ping(`${ML_URL}/health`),
        pingAI(),
      ]);
      if (!cancelled) {
        setBackend(be ? 'ok' : 'down');
        setMl(mlOk ? 'ok' : 'down');
        setAi(aiResult.ok ? 'ok' : 'down');
        if (!aiResult.ok) {
          setNeo4j('down');
        } else if (aiResult.neo4j) {
          setNeo4j('ok');
        } else {
          setNeo4j('down');
        }
      }
    };
    check();
    const id = setInterval(check, 60000);
    return () => {
      cancelled = true;
      clearInterval(id);
    };
  }, []);

  const dot = (status: ServiceStatus) => {
    if (status === 'loading') return 'bg-slate-400';
    return status === 'ok' ? 'bg-emerald-500' : 'bg-red-500';
  };

  return (
    <div
      className="hidden items-center gap-3 rounded-lg border border-slate-200 bg-slate-50 px-3 py-1.5 text-xs dark:border-navy-600 dark:bg-navy-800/60 md:flex"
      title="État des services"
    >
      <span className="flex items-center gap-1.5 text-slate-600 dark:text-slate-300">
        <span className={`h-2 w-2 rounded-full ${dot(backend)}`} />
        API
      </span>
      <span className="flex items-center gap-1.5 text-slate-600 dark:text-slate-300">
        <span className={`h-2 w-2 rounded-full ${dot(ml)}`} />
        ML
      </span>
      <span className="flex items-center gap-1.5 text-slate-600 dark:text-slate-300">
        <span className={`h-2 w-2 rounded-full ${dot(ai)}`} />
        IA
      </span>
      <span className="flex items-center gap-1.5 text-slate-600 dark:text-slate-300">
        <span className={`h-2 w-2 rounded-full ${dot(neo4j)}`} />
        Neo4j
      </span>
    </div>
  );
}
