import { Component, type ReactNode } from 'react';

interface Props {
  children: ReactNode;
}

interface State {
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  render() {
    if (this.state.error) {
      return (
        <div className="max-w-lg mx-auto mt-16 card border-red-500/30 bg-red-500/5">
          <h1 className="font-display text-xl font-bold text-red-300 mb-2">Erreur interface</h1>
          <p className="text-sm text-slate-400 mb-5">{this.state.error.message}</p>
          <button
            type="button"
            onClick={() => window.location.reload()}
            className="btn-primary text-sm"
          >
            Recharger la page
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
