import { useState, useEffect } from 'react';
import { Link } from 'react-router';
import { api } from '../services/api';

export default function SettingsPage() {
  const [settings, setSettings] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    try {
      setLoading(true);
      const response = await api.getSettings();
      setSettings(response);
    } catch (err: any) {
      setError(err.message || 'Failed to load settings');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-background">
      <header className="bg-surface shadow-sm">
        <div className="max-w-7xl mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <h1 className="text-2xl font-heading text-primary">Settings</h1>
            <nav className="flex gap-4">
              <Link to="/books" className="text-primary hover:underline">Books</Link>
              <Link to="/wishlist" className="text-primary hover:underline">Wishlist</Link>
              <button
                onClick={async () => {
                  await api.logout();
                  window.location.href = '/';
                }}
                className="text-error hover:underline"
              >
                Logout
              </button>
            </nav>
          </div>
        </div>
      </header>

      <main className="max-w-4xl mx-auto px-4 py-8">
        {loading ? (
          <div className="text-center py-12">
            <p className="text-text-light">Loading settings...</p>
          </div>
        ) : error ? (
          <div className="text-center py-12">
            <p className="text-error">{error}</p>
          </div>
        ) : (
          <div className="bg-surface rounded-lg shadow-md p-6">
            <h2 className="font-heading text-xl text-primary mb-4">Site Settings</h2>
            <div className="space-y-4">
              {Object.entries(settings).map(([key, value]) => (
                <div key={key} className="flex justify-between items-center py-2 border-b border-secondary/10">
                  <span className="font-medium text-text">{key}</span>
                  <span className="text-text-light">{value}</span>
                </div>
              ))}
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
