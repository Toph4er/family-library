import { useState } from 'react';
import { useNavigate, Link } from 'react-router';
import { api } from '../services/api';
import { useAuth } from '../context/AuthContext';

export default function LoginPage() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const { refreshUser } = useAuth();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await api.login(username, password);
      await refreshUser();
      navigate('/books');
    } catch (err: any) {
      setError(err.message || 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center relative overflow-hidden">
      <span id="main-content" className="sr-only" />
      {/* Background decorative vines */}
      <div className="absolute inset-0 pointer-events-none overflow-hidden" aria-hidden="true">
        <svg className="absolute -top-12 -right-12 w-48 h-48 text-primary/5" viewBox="0 0 200 200" fill="currentColor">
          <path d="M200 0 C160 30 120 60 100 100 C80 140 60 170 0 200" stroke="currentColor" strokeWidth="3" fill="none" />
          <path d="M160 40 C140 30 120 50 130 70 C140 90 160 80 170 60" fill="currentColor" opacity="0.5" />
          <path d="M120 90 C100 80 80 100 90 120 C100 140 120 130 130 110" fill="currentColor" opacity="0.5" />
        </svg>
        <svg className="absolute -bottom-12 -left-12 w-48 h-48 text-primary/5" viewBox="0 0 200 200" fill="currentColor" style={{ transform: 'rotate(180deg)' }}>
          <path d="M200 0 C160 30 120 60 100 100 C80 140 60 170 0 200" stroke="currentColor" strokeWidth="3" fill="none" />
          <path d="M160 40 C140 30 120 50 130 70 C140 90 160 80 170 60" fill="currentColor" opacity="0.5" />
          <path d="M120 90 C100 80 80 100 90 120 C100 140 120 130 130 110" fill="currentColor" opacity="0.5" />
        </svg>
      </div>

      <main role="main" className="w-full max-w-md p-8 bg-surface/95 backdrop-blur-sm rounded-2xl shadow-xl border border-secondary/10 animate-fade-in-up relative">
        {/* Decorative top leaf accent */}
        <div className="absolute -top-3 left-1/2 -translate-x-1/2" aria-hidden="true">
          <svg className="w-6 h-6 text-primary/30 rotate-180" viewBox="0 0 24 24" fill="currentColor">
            <path d="M17 8C8 10 5.9 16.17 3.82 21.34l1.89.66.95-2.3c.48.17.98.3 1.34.3C19 20 22 3 22 3c-1 2-8 2.25-13 3.25S2 11.5 2 13.5s1.75 3.75 1.75 3.75C7 8 17 8 17 8z" />
          </svg>
        </div>

        {/* Header */}
        <div className="text-center mb-6">
          {/* Key icon */}
          <div className="inline-flex items-center justify-center w-14 h-14 rounded-full bg-primary/10 mb-4" aria-hidden="true">
            <svg className="w-7 h-7 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
              <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4" />
            </svg>
          </div>
          <h1 className="text-3xl font-heading text-primary">Admin Login</h1>
          <p className="text-text-light mt-1 font-accent text-lg">Welcome back, keeper of the library</p>
        </div>

        {error && (
          <div className="mb-4 p-3 bg-error/10 border border-error/20 rounded-xl text-error text-sm flex items-center gap-2" role="alert">
            <svg className="w-4 h-4 flex-shrink-0" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
              <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z" />
            </svg>
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4" noValidate>
          <div>
            <label htmlFor="username" className="block text-sm font-medium text-text mb-1.5">
              Username
            </label>
            <div className="relative">
              <svg className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-light/40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
                <circle cx="12" cy="7" r="4" />
              </svg>
              <input
                id="username"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="w-full pl-10 pr-3 py-2.5 border border-secondary/20 rounded-xl bg-surface text-text placeholder:text-text-light/40 focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary/50 transition-all duration-200"
                required
                autoFocus
                placeholder="Enter your username"
                autoComplete="username"
                aria-required="true"
              />
            </div>
          </div>

          <div>
            <label htmlFor="password" className="block text-sm font-medium text-text mb-1.5">
              Password
            </label>
            <div className="relative">
              <svg className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-light/40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
                <path d="M7 11V7a5 5 0 0 1 10 0v4" />
              </svg>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full pl-10 pr-3 py-2.5 border border-secondary/20 rounded-xl bg-surface text-text placeholder:text-text-light/40 focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary/50 transition-all duration-200"
                required
                placeholder="Enter your password"
                autoComplete="current-password"
                aria-required="true"
              />
            </div>
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full py-3 px-4 bg-primary text-white rounded-xl
                       shadow-lg shadow-primary/20 hover:shadow-xl hover:shadow-primary/30
                       hover:-translate-y-0.5 active:translate-y-0
                       font-medium text-base
                       focus:outline-none focus:ring-2 focus:ring-primary/50
                       disabled:opacity-50 disabled:cursor-not-allowed
                       transition-all duration-200 ease-in-out"
            aria-label={loading ? 'Signing in, please wait' : 'Sign in to your admin account'}
          >
            {loading ? (
              <span className="flex items-center justify-center gap-2">
                <svg className="w-4 h-4 animate-spin" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                  <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="3" strokeDasharray="4 4" />
                </svg>
                Signing in...
              </span>
            ) : 'Sign In'}
          </button>
        </form>

        {/* Divider */}
        <div className="vine-divider mt-6 mb-4" aria-hidden="true">
          <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor">
            <path d="M6 1 C6 1 4 4 4 6 C4 7.1 4.9 8 6 8 C7.1 8 8 7.1 8 6 C8 4 6 1 6 1Z" />
          </svg>
        </div>

        <div className="text-center">
          <Link to="/guest-login" className="text-primary hover:text-text-light transition-colors text-sm font-medium" aria-label="Sign in as a guest instead">
            Guest access? Sign in as guest →
          </Link>
        </div>
      </main>
    </div>
  );
}
