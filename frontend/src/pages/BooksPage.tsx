import { useState, useEffect } from 'react';
import { Link } from 'react-router';
import { api } from '../services/api';
import type { Book, CreateBookRequest } from '../types/book';
import Modal from '../components/ui/Modal';
import BookForm from '../components/books/BookForm';

export default function BooksPage() {
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

  const handleCreateBook = async (data: CreateBookRequest) => {
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
      <header className="bg-surface shadow-sm">
        <div className="max-w-7xl mx-auto px-4 py-4">
          <div className="flex items-center justify-between flex-wrap gap-4">
            <h1 className="text-2xl font-heading text-primary">Book Collection</h1>
            <div className="flex items-center gap-4">
              <button
                onClick={() => setShowAddModal(true)}
                className="px-4 py-2 bg-primary text-white rounded-md hover:bg-primary/90 transition-colors font-medium text-sm"
              >
                + Add New Book
              </button>
              <nav className="flex gap-4">
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
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 py-8">
        <div className="mb-6">
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search books..."
            className="w-full px-4 py-2 border border-secondary/30 rounded-md focus:outline-none focus:ring-2 focus:ring-primary/50"
          />
        </div>

        {loading ? (
          <div className="text-center py-12">
            <p className="text-text-light">Loading books...</p>
          </div>
        ) : error ? (
          <div className="text-center py-12">
            <p className="text-error">{error}</p>
          </div>
        ) : books.length === 0 ? (
          <div className="text-center py-12">
            <p className="text-text-light mb-4">No books in the collection yet.</p>
            <p className="text-sm text-text-light">Add your first book to get started!</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {books.map((book) => (
              <Link
                key={book.id}
                to={`/books/${book.id}`}
                className="bg-surface rounded-lg shadow-md hover:shadow-lg transition-shadow overflow-hidden"
              >
                {book.cover_image_url && (
                  <img
                    src={book.cover_image_url}
                    alt={book.title}
                    className="w-full h-48 object-cover"
                  />
                )}
                <div className="p-4">
                  <h3 className="font-heading text-lg text-primary">{book.title}</h3>
                  {book.authors && (
                    <p className="text-text-light text-sm mt-1">{book.authors}</p>
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
