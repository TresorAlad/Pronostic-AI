import { useState } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { api } from '../api';
import Logo from '../components/Logo';
import { useAuth } from '../hooks/useAuth';

export default function Auth() {
  const [isLogin, setIsLogin] = useState(true);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const from = (location.state as { from?: string } | null)?.from ?? '/account';
  const { login, isAuthenticated } = useAuth();

  if (isAuthenticated) {
    return (
      <div className="card mx-auto max-w-md text-center py-10">
        <p className="text-slate-500">Vous êtes déjà connecté.</p>
        <Link to="/account" className="btn-primary mt-4 inline-block">
          Voir mon compte
        </Link>
      </div>
    );
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);
    try {
      const result = isLogin
        ? await api.login(email, password)
        : await api.register(email, password, displayName);
      login(result.token, result.user);
      navigate(from, { replace: true });
    } catch {
      setError('Erreur de connexion. Vérifiez vos identifiants.');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="max-w-md mx-auto mt-8">
      <div className="mb-8 flex justify-center">
        <Logo size="lg" />
      </div>
      <div className="card">
        <h1 className="font-display text-2xl font-bold mb-2 text-center text-heading">
          {isLogin ? 'Connexion' : 'Inscription'}
        </h1>
        <p className="text-sm text-slate-500 text-center mb-6">
          {isLogin
            ? 'Accédez à vos coupons sauvegardés et à votre historique.'
            : 'Créez un compte pour sauvegarder vos coupons IA.'}
        </p>
        <form onSubmit={handleSubmit} className="space-y-4">
          {!isLogin && (
            <input
              type="text"
              placeholder="Nom d'affichage"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              className="input-field"
            />
          )}
          <input
            type="email"
            placeholder="Email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            className="input-field"
          />
          <input
            type="password"
            placeholder="Mot de passe"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            className="input-field"
          />
          {error && <p className="text-red-400 text-sm">{error}</p>}
          <button type="submit" disabled={submitting} className="btn-primary w-full">
            {submitting ? 'Connexion...' : isLogin ? 'Se connecter' : "S'inscrire"}
          </button>
        </form>
        <button
          onClick={() => setIsLogin(!isLogin)}
          className="text-sm text-slate-400 hover:text-brand-dark dark:hover:text-brand-light mt-5 w-full text-center transition-colors"
        >
          {isLogin ? "Pas de compte ? S'inscrire" : 'Déjà un compte ? Se connecter'}
        </button>
      </div>
    </div>
  );
}
