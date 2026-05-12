import { useState, useEffect } from 'react';
import { Link } from 'react-router';
import { api } from '../services/api';
import type { WishlistItem } from '../types/wishlist';

// Priority badge component
const PriorityBadge = ({ priority }: { priority: number }) => {
  // Color coding: high priority (7-10) = warm orange, medium (4-6) = amber, low (1-3) = green
  let colorClass = 'bg-success/15 text-success border-success/20';
  let dotClass = 'bg-success';
  if (priority >= 7) {
    colorClass = 'bg-warning/15 text-warning border-warning/20';
    dotClass = 'bg-warning';
  } else if (priority >= 4) {
    colorClass = 'bg-accent/20 text-secondary border-accent/30';
    dotClass = 'bg-accent';
  }

  return (
    <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border ${colorClass}`}>
      <span className={`w-1.5 h-1.5 rounded-full ${dotClass}`} />
      {priority}/10
    </span>
  );
};

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

  // Sort: unfulfilled first, then by priority descending
  const sortedItems = [...items].sort((a, b) => {
    if (a.fulfilled !== b.fulfilled) return a.fulfilled ? 1 : -1;
    return b.priority - a.priority;
  });

  return (
    <div className="min-h-screen bg-background">
      <header className="bg-surface/90 backdrop-blur-sm shadow-md border-b border-secondary/10">
        <div className="max-w-7xl mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              {/* Star icon */}
              <svg className="w-6 h-6 text-accent" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
              </svg>
              <h1 className="text-2xl font-heading text-primary">Wishlist</h1>
            </div>
            <nav className="flex gap-4">
              <Link to="/books" className="text-primary hover:text-text-light transition-colors font-medium text-sm">Books</Link>
              <Link to="/settings" className="text-primary hover:text-text-light transition-colors font-medium text-sm">Settings</Link>
              <button
                onClick={async () => {
                  await api.logout();
                  window.location.href = '/';
                }}
                className="text-error hover:text-error/80 transition-colors font-medium text-sm"
              >
                Logout
              </button>
            </nav>
          </div>
        </div>
      </header>

      <main className="max-w-4xl mx-auto px-4 py-8">
        {loading ? (
          <div className="text-center py-16 animate-gentle-pulse">
            <svg className="w-12 h-12 mx-auto text-primary/30 animate-spin" viewBox="0 0 40 40" fill="none">
              <circle cx="20" cy="20" r="18" stroke="currentColor" strokeWidth="2" strokeDasharray="4 4" />
              <circle cx="20" cy="20" r="12" stroke="currentColor" strokeWidth="1.5" strokeDasharray="3 3" />
              <circle cx="20" cy="20" r="6" stroke="currentColor" strokeWidth="1" strokeDasharray="2 2" />
            </svg>
            <p className="text-text-light mt-4 font-accent text-xl">Checking the wish list...</p>
          </div>
        ) : error ? (
          <div className="text-center py-12">
            <p className="text-error">{error}</p>
          </div>
        ) : sortedItems.length === 0 ? (
          <div className="text-center py-16 animate-fade-in">
            <svg className="w-16 h-16 mx-auto text-accent/20 mb-4" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
            </svg>
            <p className="text-text-light mb-2 font-accent text-2xl">No items on the wishlist yet.</p>
            <p className="text-sm text-text-light/70">Add books you'd love to have!</p>
          </div>
        ) : (
          <div className="space-y-4 stagger-children">
            {sortedItems.map((item) => (
              <div
                key={item.id}
                className={`bg-surface rounded-xl shadow-md card-hover border transition-all duration-200 ${
                  item.fulfilled
                    ? 'border-success/20 bg-success/[0.02]'
                    : 'border-secondary/5'
                }`}
              >
                <div className="p-6">
                  <div className="flex justify-between items-start gap-4">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-3 flex-wrap">
                        <h3 className={`font-heading text-lg ${
                          item.fulfilled ? 'text-text-light line-through' : 'text-primary'
                        }`}>
                          {item.title}
                        </h3>
                        {item.fulfilled && (
                          <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-success/15 text-success rounded-full text-xs font-medium">
                            <svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                              <polyline points="20 6 9 17 4 12" />
                            </svg>
                            Fulfilled
                          </span>
                        )}
                      </div>
                      {item.author && (
                        <p className={`text-sm mt-1 ${item.fulfilled ? 'text-text-light/50' : 'text-text-light'}`}>
                          by {item.author}
                        </p>
                      )}
                      {item.reason && (
                        <p className={`text-sm mt-2 ${item.fulfilled ? 'text-text-light/50' : 'text-text'}`}>
                          {item.reason}
                        </p>
                      )}
                    </div>
                    <div className="flex-shrink-0">
                      <PriorityBadge priority={item.priority} />
                    </div>
                  </div>

                  {/* Purchase links */}
                  {(item.amazon_url || item.thriftbooks_url) && (
                    <div className="mt-4 flex flex-wrap gap-3">
                      {item.amazon_url && (
                        <a
                          href={item.amazon_url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-primary/10 text-primary rounded-lg text-sm font-medium hover:bg-primary/20 transition-colors"
                        >
                          <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                            <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
                            <polyline points="15 3 21 3 21 9" />
                            <line x1="10" y1="14" x2="21" y2="3" />
                          </svg>
                          Amazon
                        </a>
                      )}
                      {item.thriftbooks_url && (
                        <a
                          href={item.thriftbooks_url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-secondary/10 text-secondary rounded-lg text-sm font-medium hover:bg-secondary/20 transition-colors"
                        >
                          <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                            <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
                            <polyline points="15 3 21 3 21 9" />
                            <line x1="10" y1="14" x2="21" y2="3" />
                          </svg>
                          ThriftBooks
                        </a>
                      )}
                    </div>
                  )}

                  {/* Fulfilled date */}
                  {item.fulfilled && item.fulfilled_at && (
                    <p className="mt-2 text-xs text-success/70">
                      ✓ Fulfilled on {new Date(item.fulfilled_at).toLocaleDateString()}
                    </p>
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
