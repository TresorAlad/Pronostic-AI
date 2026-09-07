import { useEffect, useId, useRef, useState } from 'react';

export type FilterOption = {
  value: string;
  label: string;
  hint?: 'live' | 'scheduled' | 'finished';
};

type FilterSelectProps = {
  label: string;
  value: string;
  options: FilterOption[];
  onChange: (value: string) => void;
  placeholder?: string;
};

function StatusDot({ hint }: { hint?: FilterOption['hint'] }) {
  if (!hint || hint === 'finished') {
    return <span className="filter-dot filter-dot-muted" />;
  }
  if (hint === 'live') {
    return <span className="filter-dot filter-dot-live" />;
  }
  return <span className="filter-dot filter-dot-scheduled" />;
}

export default function FilterSelect({
  label,
  value,
  options,
  onChange,
  placeholder = 'Choisir…',
}: FilterSelectProps) {
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const listId = useId();
  const selected = options.find((o) => o.value === value);

  useEffect(() => {
    const onPointerDown = (event: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', onPointerDown);
    return () => document.removeEventListener('mousedown', onPointerDown);
  }, []);

  return (
    <div className="filter-field" ref={rootRef}>
      <span className="filter-label">{label}</span>
      <button
        type="button"
        className={`filter-trigger ${open ? 'filter-trigger-open' : ''}`}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={listId}
        onClick={() => setOpen((v) => !v)}
      >
        <span className="filter-trigger-value">
          {selected ? (
            <>
              <StatusDot hint={selected.hint} />
              {selected.label}
            </>
          ) : (
            placeholder
          )}
        </span>
        <Chevron open={open} />
      </button>

      {open && (
        <ul id={listId} className="filter-menu" role="listbox">
          {options.map((option) => {
            const active = option.value === value;
            return (
              <li key={option.value} role="option" aria-selected={active}>
                <button
                  type="button"
                  className={`filter-option ${active ? 'filter-option-active' : ''}`}
                  onClick={() => {
                    onChange(option.value);
                    setOpen(false);
                  }}
                >
                  <span className="filter-option-left">
                    <StatusDot hint={option.hint} />
                    {option.label}
                  </span>
                  {active && <CheckIcon />}
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

type FilterDateProps = {
  label: string;
  value: string;
  onChange: (value: string) => void;
};

export function FilterDate({ label, value, onChange }: FilterDateProps) {
  return (
    <div className="filter-field">
      <span className="filter-label">{label}</span>
      <div className="filter-date-wrap">
        <CalendarIcon />
        <input
          type="date"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="filter-date"
        />
        {value && (
          <button
            type="button"
            className="filter-date-clear"
            aria-label="Effacer la date"
            onClick={() => onChange('')}
          >
            ×
          </button>
        )}
      </div>
    </div>
  );
}

function Chevron({ open }: { open: boolean }) {
  return (
    <svg
      className={`filter-chevron ${open ? 'filter-chevron-open' : ''}`}
      viewBox="0 0 20 20"
      fill="currentColor"
      aria-hidden
    >
      <path
        fillRule="evenodd"
        d="M5.23 7.21a.75.75 0 011.06.02L10 10.94l3.71-3.71a.75.75 0 111.06 1.06l-4.24 4.25a.75.75 0 01-1.06 0L5.21 8.29a.75.75 0 01.02-1.08z"
        clipRule="evenodd"
      />
    </svg>
  );
}

function CheckIcon() {
  return (
    <svg className="filter-check" viewBox="0 0 20 20" fill="currentColor" aria-hidden>
      <path
        fillRule="evenodd"
        d="M16.704 5.29a1 1 0 010 1.42l-7.25 7.25a1 1 0 01-1.42 0l-3.25-3.25a1 1 0 111.42-1.42l2.54 2.54 6.54-6.54a1 1 0 011.42 0z"
        clipRule="evenodd"
      />
    </svg>
  );
}

function CalendarIcon() {
  return (
    <svg className="filter-date-icon" viewBox="0 0 20 20" fill="currentColor" aria-hidden>
      <path d="M5.25 3A2.25 2.25 0 003 5.25v9.5A2.25 2.25 0 005.25 17h9.5A2.25 2.25 0 0017 14.75v-9.5A2.25 2.25 0 0014.75 3h-9.5zM4.5 5.25a.75.75 0 01.75-.75h.75V4.5a.75.75 0 011.5 0v.75h4.5V4.5a.75.75 0 011.5 0v.75h.75a.75.75 0 01.75.75v1.5h-10.5v-1.5zm10.5 9.75a.75.75 0 01-.75.75h-9.5a.75.75 0 01-.75-.75v-6.75h10.5v6.75z" />
    </svg>
  );
}
