import { useState } from 'react';
import { useNavigate } from 'react-router';
import { api } from '../services/api';
import { useAuth } from '../context/AuthContext';

export default function GuestLoginPage() {
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
      await api.guestLogin(password);
      await refreshUser();
      navigate('/books');
    } catch (err: any) {
      setError(err.message || 'Guest login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <div className="w-full max-w-md p-8 bg-surface rounded-lg shadow-lg">
        <h1 className="text-3xl font-heading text-primary mb-2 text-center">Guest Login</h1>
        <p className="text-text-light mb-6 text-center">Browse the collection and wishlist</p>

        {error && (
          <div className="mb-4 p-3 bg-error/10 border border-error/30 rounded text-error text-sm">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="guest-password" className="block text-sm font-medium text-text mb-1">
              Guest Password
            </label>
            <input
              id="guest-password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-3 py-2 border border-secondary/30 rounded-md focus:outline-none focus:ring-2 focus:ring-primary/50"
              required
              autoFocus
              placeholder="Enter the guest password"
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full py-2 px-4 bg-secondary text-white rounded-md hover:bg-secondary/90 focus:outline-none focus:ring-2 focus:ring-secondary/50 disabled:opacity-50"
          >
            {loading ? 'Signing in...' : 'Sign In as Guest'}
          </button>
        </form>

        <div className="mt-6 text-center">
          <a href="/login" className="text-primary hover:underline text-sm">
            Admin login
          </a>
        </div>
      </div>
    </div>
  );
}
