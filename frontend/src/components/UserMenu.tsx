import { useEffect, useRef, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api';
import { useAuth, userInitials, userLabel } from '../hooks/useAuth';

export default function UserMenu() {
  const { user, loading, isAuthenticated, logout } = useAuth();
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data: notifications } = useQuery({
    queryKey: ['notifications'],
    queryFn: api.getNotifications,
    enabled: isAuthenticated,
    refetchInterval: 60000,
  });
  const unread = notifications?.filter((n) => !n.read_at).length ?? 0;

  const markRead = useMutation({
    mutationFn: api.markNotificationRead,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['notifications'] }),
  });

  useEffect(() => {
    const onPointerDown = (event: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', onPointerDown);
    return () => document.removeEventListener('mousedown', onPointerDown);
  }, []);

  if (loading) {
    return (
      <div className="user-menu-skeleton" aria-hidden>
        <span className="user-avatar user-avatar-muted">···</span>
      </div>
    );
  }

  if (!isAuthenticated || !user) {
    return (
      <Link to="/auth" className="nav-link whitespace-nowrap ml-1">
        Connexion
      </Link>
    );
  }

  const initials = userInitials(user);
  const label = userLabel(user);

  return (
    <div className="user-menu" ref={rootRef}>
      <button
        type="button"
        className={`user-menu-trigger ${open ? 'user-menu-trigger-open' : ''}`}
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        aria-haspopup="menu"
      >
        <span className="user-avatar">{initials}</span>
        <span className="user-menu-name hidden sm:inline">{label}</span>
        {unread > 0 && (
          <span className="rounded-full bg-red-500 text-white text-[10px] px-1.5 py-0.5">{unread}</span>
        )}
        <span className="user-menu-dot" title="Connecté" />
      </button>

      {open && (
        <div className="user-menu-panel" role="menu">
          <div className="user-menu-header">
            <span className="user-avatar user-avatar-lg">{initials}</span>
            <div className="min-w-0">
              <p className="font-semibold text-heading truncate">{label}</p>
              <p className="text-xs text-slate-500 truncate">{user.email}</p>
              <span className="user-status-badge">Connecté</span>
            </div>
          </div>

          {notifications && notifications.length > 0 && (
            <div className="user-menu-links border-t border-slate-200 dark:border-navy-600 pt-2 mt-2">
              <p className="text-xs font-semibold text-slate-500 px-2 mb-1">Notifications</p>
              {notifications.slice(0, 3).map((n) => (
                <button
                  key={n.id}
                  type="button"
                  className="user-menu-link text-left w-full"
                  onClick={() => {
                    if (!n.read_at) markRead.mutate(n.id);
                    setOpen(false);
                  }}
                >
                  <span className={n.read_at ? 'opacity-60' : ''}>{n.title}</span>
                </button>
              ))}
            </div>
          )}

          <div className="user-menu-links">
            <Link to="/account" className="user-menu-link" onClick={() => setOpen(false)}>
              Mon compte
            </Link>
            <Link to="/my-performance" className="user-menu-link" onClick={() => setOpen(false)}>
              Ma performance
            </Link>
            <Link to="/my-coupons" className="user-menu-link" onClick={() => setOpen(false)}>
              Mes coupons
            </Link>
            <Link to="/coupon" className="user-menu-link" onClick={() => setOpen(false)}>
              Générer un coupon
            </Link>
          </div>

          <button
            type="button"
            className="user-menu-logout"
            onClick={() => {
              logout();
              setOpen(false);
              navigate('/');
            }}
          >
            Se déconnecter
          </button>
        </div>
      )}
    </div>
  );
}
