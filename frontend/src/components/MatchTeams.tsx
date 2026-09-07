import TeamLogo from './TeamLogo';

interface Team {
  name: string;
  logo_url?: string;
}

interface MatchTeamsProps {
  home: Team;
  away: Team;
  homeScore?: number;
  awayScore?: number;
  minute?: number;
  status: string;
  layout?: 'card' | 'detail';
}

export default function MatchTeams({
  home,
  away,
  homeScore,
  awayScore,
  minute,
  status,
  layout = 'card',
}: MatchTeamsProps) {
  const logoSize = layout === 'detail' ? 'lg' : 'md';

  return (
    <div className="flex items-center justify-between gap-4">
      <div className={`flex flex-1 items-center gap-3 ${layout === 'card' ? 'flex-row-reverse justify-start' : 'flex-col'}`}>
        <TeamLogo name={home.name} logoUrl={home.logo_url} size={logoSize} />
        <span className={`font-semibold text-heading ${layout === 'detail' ? 'text-2xl' : 'text-lg'} ${layout === 'card' ? 'text-right' : 'text-center'}`}>
          {home.name}
        </span>
      </div>

      <div className="text-center px-2 shrink-0">
        {status === 'live' ? (
          <div>
            <span className={`font-bold text-brand-dark dark:text-brand-light ${layout === 'detail' ? 'text-4xl' : 'text-2xl'}`}>
              {homeScore ?? 0} - {awayScore ?? 0}
            </span>
            <p className="text-xs text-red-400 animate-pulse">{minute ?? 0}'</p>
          </div>
        ) : status === 'finished' ? (
          <span className={`font-bold ${layout === 'detail' ? 'text-4xl text-brand-dark dark:text-brand-light' : 'text-xl text-heading'}`}>
            {homeScore} - {awayScore}
          </span>
        ) : (
          <span className="text-slate-500 font-medium text-sm">VS</span>
        )}
      </div>

      <div className={`flex flex-1 items-center gap-3 ${layout === 'card' ? '' : 'flex-col'}`}>
        <TeamLogo name={away.name} logoUrl={away.logo_url} size={logoSize} />
        <span className={`font-semibold text-heading ${layout === 'detail' ? 'text-2xl' : 'text-lg'} ${layout === 'card' ? '' : 'text-center'}`}>
          {away.name}
        </span>
      </div>
    </div>
  );
}
