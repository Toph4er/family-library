import { useState, useEffect } from 'react';
import { Link } from 'react-router';
import { api } from '../services/api';
import type { WishlistItem } from '../types/wishlist';

export default function WishlistPage() {
  const [items, setItems] = useState<WishlistItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    loadWishlist();
  }, []);

  const loadWishlist = async () => {
    try {
      setLoading(true);
      const response = await api.getWishlist();
      setItems(response.items || []);
    } catch (err: any) {
      setError(err.message || 'Failed to load wishlist');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-background">
      <header className="bg-surface shadow-sm">
        <div className="max-w-7xl mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <h1 className="text-2xl font-heading text-primary">Wishlist</h1>
            <nav className="flex gap-4">
              <Link to="/books" className="text-primary hover:underline">Books</Link>
              <Link to="/settings" className="text-primary hover:underline">Settings</Link>
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

      <main className="max-w-7xl mx-auto px-4 py-8">
        {loading ? (
          <div className="text-center py-12">
            <p className="text-text-light">Loading wishlist...</p>
          </div>
        ) : error ? (
          <div className="text-center py-12">
            <p className="text-error">{error}</p>
          </div>
        ) : items.length === 0 ? (
          <div className="text-center py-12">
            <p className="text-text-light">No items on the wishlist yet.</p>
          </div>
        ) : (
          <div className="space-y-4">
            {items.map((item) => (
              <div
                key={item.id}
                className={`bg-surface rounded-lg shadow-md p-6 ${item.fulfilled ? 'opacity-60' : ''}`}
              >
                <div className="flex justify-between items-start">
                  <div>
                    <h3 className="font-heading text-lg text-primary">{item.title}</h3>
                    {item.author && (
                      <p className="text-text-light text-sm mt-1">by {item.author}</p>
                    )}
                    {item.reason && (
                      <p className="text-sm mt-2">{item.reason}</p>
                    )}
                  </div>
                  <span className="text-sm text-text-light">Priority: {item.priority}/10</span>
                </div>
                <div className="mt-4 flex gap-2">
                  {item.amazon_url && (
                    <a
                      href={item.amazon_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-primary hover:underline text-sm"
                    >
                      Amazon
                    </a>
                  )}
                  {item.thriftbooks_url && (
                    <a
                      href={item.thriftbooks_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-primary hover:underline text-sm"
                    >
                      ThriftBooks
                    </a>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </main>
    </div>
  );
}
