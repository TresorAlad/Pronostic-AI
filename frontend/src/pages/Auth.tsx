import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api';

export default function Auth() {
  const [isLogin, setIsLogin] = useState(true);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [error, setError] = useState('');
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      const result = isLogin
        ? await api.login(email, password)
        : await api.register(email, password, displayName);
      localStorage.setItem('token', result.token);
      navigate('/');
    } catch {
      setError('Erreur de connexion. Verifiez vos identifiants.');
    }
  };

  return (
    <div className="max-w-md mx-auto mt-12">
      <div className="card">
        <h1 className="text-2xl font-bold mb-6 text-center">
          {isLogin ? 'Connexion' : 'Inscription'}
        </h1>
        <form onSubmit={handleSubmit} className="space-y-4">
          {!isLogin && (
            <input
              type="text"
              placeholder="Nom d'affichage"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              className="w-full bg-pitch-900 border border-pitch-700 rounded-lg px-4 py-2"
            />
          )}
          <input
            type="email"
            placeholder="Email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            className="w-full bg-pitch-900 border border-pitch-700 rounded-lg px-4 py-2"
          />
          <input
            type="password"
            placeholder="Mot de passe"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            className="w-full bg-pitch-900 border border-pitch-700 rounded-lg px-4 py-2"
          />
          {error && <p className="text-red-400 text-sm">{error}</p>}
          <button type="submit" className="btn-primary w-full">
            {isLogin ? 'Se connecter' : "S'inscrire"}
          </button>
        </form>
        <button
          onClick={() => setIsLogin(!isLogin)}
          className="text-sm text-gray-400 hover:text-accent mt-4 w-full text-center"
        >
          {isLogin ? "Pas de compte ? S'inscrire" : 'Deja un compte ? Se connecter'}
        </button>
      </div>
    </div>
  );
}
