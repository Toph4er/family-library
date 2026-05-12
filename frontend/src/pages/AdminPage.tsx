import { Link } from 'react-router';
import { api } from '../services/api';

export default function AdminPage() {
  return (
    <div className="min-h-screen bg-background">
      <header className="bg-surface shadow-sm">
        <div className="max-w-7xl mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <h1 className="text-2xl font-heading text-primary">Admin Dashboard</h1>
            <nav className="flex gap-4">
              <Link to="/books" className="text-primary hover:underline">Books</Link>
              <Link to="/wishlist" className="text-primary hover:underline">Wishlist</Link>
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
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="bg-surface rounded-lg shadow-md p-6">
            <h2 className="font-heading text-xl text-primary mb-2">Books</h2>
            <p className="text-text-light">Manage your book collection</p>
            <Link to="/books" className="inline-block mt-4 text-primary hover:underline">
              View Collection →
            </Link>
          </div>

          <div className="bg-surface rounded-lg shadow-md p-6">
            <h2 className="font-heading text-xl text-primary mb-2">Wishlist</h2>
            <p className="text-text-light">Manage books you want to add</p>
            <Link to="/wishlist" className="inline-block mt-4 text-primary hover:underline">
              View Wishlist →
            </Link>
          </div>

          <div className="bg-surface rounded-lg shadow-md p-6">
            <h2 className="font-heading text-xl text-primary mb-2">Settings</h2>
            <p className="text-text-light">Configure your library</p>
            <Link to="/settings" className="inline-block mt-4 text-primary hover:underline">
              View Settings →
            </Link>
          </div>
        </div>
      </main>
    </div>
  );
}
