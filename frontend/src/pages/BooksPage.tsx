import { useState, useEffect } from 'react';
import { Link } from 'react-router';
import { api } from '../services/api';
import { useAuth } from '../context/AuthContext';
import type { Book, CreateBookRequest, UpdateBookRequest } from '../types/book';
import Modal from '../components/ui/Modal';
import BookForm from '../components/books/BookForm';

export default function BooksPage() {
  const { isAdmin, logout } = useAuth();
  const [books, setBooks] = useState<Book[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [showAddModal, setShowAddModal] = useState(false);

  useEffect(() => {
    loadBooks();
  }, [searchQuery]);

  const loadBooks = async () => {
    try {
      setLoading(true);
      const response = await api.getBooks(searchQuery ? { query: searchQuery } : undefined);
      setBooks(response.items || []);
    } catch (err: any) {
      setError(err.message || 'Failed to load books');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateBook = async (data: CreateBookRequest | UpdateBookRequest) => {
    try {
      await api.createBook(data);
      setShowAddModal(false);
      await loadBooks();
    } catch (err: any) {
      setError(err.message || 'Failed to add book');
    }
  };

  return (
    <div className="min-h-screen bg-background">
      <header className="bg-surface/90 backdrop-blur-sm shadow-md border-b border-secondary/10">
        <div className="max-w-7xl mx-auto px-4 py-4">
          <div className="flex items-center justify-between flex-wrap gap-4">
            <div className="flex items-center gap-3">
              {/* Book icon */}
              <svg className="w-7 h-7 text-primary" viewBox="0 0 24 24" fill="currentColor">
                <path d="M21 5c-1.11-.35-2.33-.5-3.5-.5-1.95 0-4.05.4-5.5 1.5-1.45-1.1-3.55-1.5-5.5-1.5S2.45 4.9 1 6v14.65c0 .25.25.5.5.5.1 0 .15-.05.25-.05C3.1 20.45 5.05 20 6.5 20c1.95 0 4.05.4 5.5 1.5 1.35-.85 3.8-1.5 5.5-1.5 1.65 0 3.35.3 4.75 1.05.1.05.15.05.25.05.25 0 .5-.25.5-.5V6c-.6-.45-1.25-.75-2-1zm0 13.5c-1.1-.35-2.3-.5-3.5-.5-1.7 0-4.15.65-5.5 1.5V8c1.35-.85 3.8-1.5 5.5-1.5 1.2 0 2.4.15 3.5.5v11.5z" />
              </svg>
              <h1 className="text-2xl font-heading text-primary">Book Collection</h1>
            </div>
            <div className="flex items-center gap-4">
              {isAdmin && (
                <button
                  onClick={() => setShowAddModal(true)}
                  className="px-5 py-2.5 bg-primary text-white rounded-xl
                             shadow-md shadow-primary/20 hover:shadow-lg hover:shadow-primary/30
                             hover:-translate-y-0.5 active:translate-y-0
                             font-medium text-sm transition-all duration-200 ease-in-out
                             flex items-center gap-2"
                >
                  <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round">
                    <path d="M12 5v14M5 12h14" />
                  </svg>
                  Add New Book
                </button>
              )}
              <nav className="flex gap-4">
                <Link to="/wishlist" className="text-primary hover:text-text-light transition-colors font-medium text-sm">Wishlist</Link>
                {isAdmin && <Link to="/settings" className="text-primary hover:text-text-light transition-colors font-medium text-sm">Settings</Link>}
                <button
                  onClick={async () => {
                    await logout();
                    window.location.href = '/';
                  }}
                  className="text-error hover:text-error/80 transition-colors font-medium text-sm"
                >
                  Logout
                </button>
              </nav>
            </div>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 py-8">
        {/* Search bar with woodland styling */}
        <div className="mb-8 animate-fade-in-up">
          <div className="relative">
            <svg className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-text-light/50" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="11" cy="11" r="8" />
              <path d="M21 21l-4.35-4.35" />
            </svg>
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search the forest of books..."
              className="w-full pl-12 pr-4 py-3 bg-surface border border-secondary/20 rounded-xl
                         text-text placeholder:text-text-light/40
                         focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary/50
                         shadow-sm hover:shadow-md transition-all duration-200"
            />
          </div>
        </div>

        {loading ? (
          <div className="text-center py-16 animate-gentle-pulse">
            {/* Tree ring loading spinner */}
            <svg className="w-12 h-12 mx-auto text-primary/30 animate-spin" viewBox="0 0 40 40" fill="none">
              <circle cx="20" cy="20" r="18" stroke="currentColor" strokeWidth="2" strokeDasharray="4 4" />
              <circle cx="20" cy="20" r="12" stroke="currentColor" strokeWidth="1.5" strokeDasharray="3 3" />
              <circle cx="20" cy="20" r="6" stroke="currentColor" strokeWidth="1" strokeDasharray="2 2" />
            </svg>
            <p className="text-text-light mt-4 font-accent text-xl">Searching the library...</p>
          </div>
        ) : error ? (
          <div className="text-center py-12">
            <p className="text-error">{error}</p>
          </div>
        ) : books.length === 0 ? (
          <div className="text-center py-16 animate-fade-in">
            <svg className="w-16 h-16 mx-auto text-primary/20 mb-4" viewBox="0 0 24 24" fill="currentColor">
              <path d="M21 5c-1.11-.35-2.33-.5-3.5-.5-1.95 0-4.05.4-5.5 1.5-1.45-1.1-3.55-1.5-5.5-1.5S2.45 4.9 1 6v14.65c0 .25.25.5.5.5.1 0 .15-.05.25-.05C3.1 20.45 5.05 20 6.5 20c1.95 0 4.05.4 5.5 1.5 1.35-.85 3.8-1.5 5.5-1.5 1.65 0 3.35.3 4.75 1.05.1.05.15.05.25.05.25 0 .5-.25.5-.5V6c-.6-.45-1.25-.75-2-1zm0 13.5c-1.1-.35-2.3-.5-3.5-.5-1.7 0-4.15.65-5.5 1.5V8c1.35-.85 3.8-1.5 5.5-1.5 1.2 0 2.4.15 3.5.5v11.5z" />
            </svg>
            <p className="text-text-light mb-2 font-accent text-2xl">No books in the collection yet.</p>
            <p className="text-sm text-text-light/70">Add your first book to get started!</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6 stagger-children">
            {books.map((book) => (
              <Link
                key={book.id}
                to={`/books/${book.id}`}
                className="group bg-surface rounded-xl shadow-md card-hover overflow-hidden border border-secondary/5"
              >
                {book.cover_image_url ? (
                  <div className="relative overflow-hidden">
                    <img
                      src={book.cover_image_url}
                      alt={book.title}
                      className="w-full h-52 object-cover group-hover:scale-105 transition-transform duration-300"
                    />
                    <div className="absolute inset-0 bg-gradient-to-t from-black/20 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
                  </div>
                ) : (
                  <div className="w-full h-32 bg-gradient-to-br from-primary/5 to-secondary/5 flex items-center justify-center">
                    <svg className="w-12 h-12 text-primary/20" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M21 5c-1.11-.35-2.33-.5-3.5-.5-1.95 0-4.05.4-5.5 1.5-1.45-1.1-3.55-1.5-5.5-1.5S2.45 4.9 1 6v14.65c0 .25.25.5.5.5.1 0 .15-.05.25-.05C3.1 20.45 5.05 20 6.5 20c1.95 0 4.05.4 5.5 1.5 1.35-.85 3.8-1.5 5.5-1.5 1.65 0 3.35.3 4.75 1.05.1.05.15.05.25.05.25 0 .5-.25.5-.5V6c-.6-.45-1.25-.75-2-1zm0 13.5c-1.1-.35-2.3-.5-3.5-.5-1.7 0-4.15.65-5.5 1.5V8c1.35-.85 3.8-1.5 5.5-1.5 1.2 0 2.4.15 3.5.5v11.5z" />
                    </svg>
                  </div>
                )}
                <div className="p-5">
                  <h3 className="font-heading text-lg text-primary group-hover:text-text transition-colors leading-snug">
                    {book.title}
                  </h3>
                  {book.authors && (
                    <p className="text-text-light text-sm mt-1.5">{book.authors}</p>
                  )}
                  {book.child_rating != null && book.child_rating > 0 && (
                    <div className="flex gap-0.5 mt-2">
                      {[1, 2, 3, 4, 5].map((star) => (
                        <span key={star} className={`text-sm ${star <= book.child_rating! ? 'text-accent' : 'text-secondary/15'}`}>
                          ★
                        </span>
                      ))}
                    </div>
                  )}
                </div>
              </Link>
            ))}
          </div>
        )}
      </main>

      {/* Add Book Modal */}
      <Modal
        isOpen={showAddModal}
        onClose={() => setShowAddModal(false)}
        title="Add New Book"
        size="xl"
      >
        <BookForm
          onSubmit={handleCreateBook}
          onCancel={() => setShowAddModal(false)}
        />
      </Modal>
    </div>
  );
}
