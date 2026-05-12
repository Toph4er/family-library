import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router';
import { api } from '../services/api';
import type { Book, UpdateBookRequest } from '../types/book';
import Modal from '../components/ui/Modal';
import BookForm from '../components/books/BookForm';

export default function BookDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [book, setBook] = useState<Book | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showEditModal, setShowEditModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    if (id) {
      loadBook(Number(id));
    }
  }, [id]);

  const loadBook = async (bookId: number) => {
    try {
      setLoading(true);
      const response = await api.getBook(bookId);
      setBook(response);
    } catch (err: any) {
      setError(err.message || 'Failed to load book');
    } finally {
      setLoading(false);
    }
  };

  const handleUpdateBook = async (data: UpdateBookRequest) => {
    if (!id) return;
    try {
      await api.updateBook(Number(id), data);
      setShowEditModal(false);
      await loadBook(Number(id));
    } catch (err: any) {
      setError(err.message || 'Failed to update book');
    }
  };

  const handleDeleteBook = async () => {
    if (!id) return;
    try {
      setDeleting(true);
      await api.deleteBook(Number(id));
      window.location.href = '/books';
    } catch (err: any) {
      setError(err.message || 'Failed to delete book');
      setDeleting(false);
    }
  };

  const renderStars = (rating: number) => {
    if (!rating || rating === 0) return null;
    return (
      <div className="flex gap-1 mt-2">
        {[1, 2, 3, 4, 5].map((star) => (
          <span key={star} className={`text-xl ${star <= rating ? 'text-accent' : 'text-secondary/20'}`}>
            ★
          </span>
        ))}
      </div>
    );
  };

  const renderTags = (value: string | undefined) => {
    if (!value) return null;
    try {
      const items = JSON.parse(value);
      if (!Array.isArray(items) || items.length === 0) return null;
      return (
        <div className="flex flex-wrap gap-2 mt-1">
          {items.map((item: string) => (
            <span
              key={item}
              className="px-2 py-0.5 bg-primary/10 text-primary text-xs rounded-full"
            >
              {item.replace(/_/g, ' ')}
            </span>
          ))}
        </div>
      );
    } catch {
      // Not JSON, render as plain text
      return <span className="text-sm">{value}</span>;
    }
  };

  const renderField = (label: string, value: string | number | undefined) => {
    if (!value) return null;
    return (
      <div>
        <span className="text-sm text-text-light">{label}:</span>
        <p className="font-medium">{value}</p>
      </div>
    );
  };

  return (
    <div className="min-h-screen bg-background">
      <header className="bg-surface shadow-sm">
        <div className="max-w-7xl mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <Link to="/books" className="text-primary hover:underline">← Back to Books</Link>
            {book && (
              <div className="flex gap-3">
                <button
                  onClick={() => setShowEditModal(true)}
                  className="px-4 py-2 border border-secondary/30 rounded-md text-text hover:bg-background transition-colors text-sm font-medium"
                >
                  Edit
                </button>
                <button
                  onClick={() => setShowDeleteModal(true)}
                  className="px-4 py-2 border border-error/30 rounded-md text-error hover:bg-error/5 transition-colors text-sm font-medium"
                >
                  Delete
                </button>
              </div>
            )}
          </div>
        </div>
      </header>

      <main className="max-w-4xl mx-auto px-4 py-8">
        {loading ? (
          <div className="text-center py-12">
            <p className="text-text-light">Loading...</p>
          </div>
        ) : error ? (
          <div className="text-center py-12">
            <p className="text-error">{error}</p>
          </div>
        ) : book ? (
          <div className="bg-surface rounded-lg shadow-md overflow-hidden">
            {book.cover_image_url && (
              <img
                src={book.cover_image_url}
                alt={book.title}
                className="w-full h-64 object-cover"
              />
            )}
            <div className="p-6">
              <h1 className="text-3xl font-heading text-primary">{book.title}</h1>
              {book.subtitle && (
                <p className="text-text-light mt-1 text-lg">{book.subtitle}</p>
              )}
              {book.authors && (
                <p className="text-text-light mt-2">by {book.authors}</p>
              )}
              {book.illustrators && (
                <p className="text-text-light mt-1">Illustrated by {book.illustrators}</p>
              )}

              {book.child_rating != null && renderStars(book.child_rating)}

              {/* Basic metadata grid */}
              <div className="mt-6 grid grid-cols-2 md:grid-cols-3 gap-4">
                {renderField('Type', book.book_type)}
                {renderField('Pages', book.page_count)}
                {renderField('Publisher', book.publisher)}
                {renderField('Year', book.publication_year)}
                {renderField('ISBN', book.isbn)}
                {renderField('Condition', book.condition)}
                {renderField('Location', book.location)}
                {renderField('Read Count', book.read_count)}
              </div>

              {/* Reading levels */}
              {book.reading_levels && (
                <div className="mt-4">
                  <span className="text-sm text-text-light">Reading Levels:</span>
                  {renderTags(book.reading_levels)}
                </div>
              )}

              {/* Genres */}
              {book.genres && (
                <div className="mt-4">
                  <span className="text-sm text-text-light">Genres:</span>
                  {renderTags(book.genres)}
                </div>
              )}

              {/* Themes */}
              {book.themes && (
                <div className="mt-4">
                  <span className="text-sm text-text-light">Themes:</span>
                  {renderTags(book.themes)}
                </div>
              )}

              {/* Awards */}
              {book.awards && (
                <div className="mt-4">
                  <span className="text-sm text-text-light">Awards:</span>
                  {renderTags(book.awards)}
                </div>
              )}

              {/* Gift info */}
              {(book.gift_from || book.gift_relationship || book.date_received) && (
                <div className="mt-6 border-t border-secondary/20 pt-4">
                  <h3 className="font-heading text-lg text-primary mb-3">Gift Information</h3>
                  <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
                    {renderField('From', book.gift_from)}
                    {renderField('Relationship', book.gift_relationship)}
                    {renderField('Date Received', book.date_received)}
                  </div>
                </div>
              )}

              {/* Notes */}
              {book.notes && (
                <div className="mt-6 border-t border-secondary/20 pt-4">
                  <h3 className="font-heading text-lg text-primary mb-2">Notes</h3>
                  <p className="text-text whitespace-pre-wrap">{book.notes}</p>
                </div>
              )}

              {/* Timestamps */}
              <div className="mt-6 border-t border-secondary/20 pt-4 text-xs text-text-light">
                <span>Added: {new Date(book.created_at).toLocaleDateString()}</span>
                {book.updated_at !== book.created_at && (
                  <span className="ml-4">Updated: {new Date(book.updated_at).toLocaleDateString()}</span>
                )}
              </div>
            </div>
          </div>
        ) : null}
      </main>

      {/* Edit Modal */}
      <Modal
        isOpen={showEditModal}
        onClose={() => setShowEditModal(false)}
        title="Edit Book"
        size="xl"
      >
        <BookForm
          book={book}
          onSubmit={handleUpdateBook}
          onCancel={() => setShowEditModal(false)}
        />
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={showDeleteModal}
        onClose={() => !deleting && setShowDeleteModal(false)}
        title="Delete Book"
        size="sm"
      >
        <p className="text-text mb-6">
          Are you sure you want to delete <strong className="text-primary">{book?.title}</strong>? This action cannot be undone.
        </p>
        <div className="flex justify-end gap-3">
          <button
            onClick={() => setShowDeleteModal(false)}
            disabled={deleting}
            className="px-4 py-2 border border-secondary/30 rounded-md text-text hover:bg-background transition-colors disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            onClick={handleDeleteBook}
            disabled={deleting}
            className="px-4 py-2 bg-error text-white rounded-md hover:bg-error/90 transition-colors font-medium disabled:opacity-50"
          >
            {deleting ? 'Deleting...' : 'Delete Book'}
          </button>
        </div>
      </Modal>
    </div>
  );
}
