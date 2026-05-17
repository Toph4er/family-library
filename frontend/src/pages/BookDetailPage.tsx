import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router';
import { api } from '../services/api';
import { useAuth } from '../context/AuthContext';
import type { Book, UpdateBookRequest } from '../types/book';
import Modal from '../components/ui/Modal';
import BookForm from '../components/books/BookForm';

// Vine divider component
const VineDivider = () => (
  <div className="vine-divider">
    <svg width="14" height="14" viewBox="0 0 14 14" fill="currentColor" aria-hidden="true">
      <path d="M7 1 C7 1 4 5 4 7 C4 8.7 5.3 10 7 10 C8.7 10 10 8.7 10 7 C10 5 7 1 7 1Z" />
    </svg>
  </div>
);

export default function BookDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { isAdmin } = useAuth();
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
      <div className="flex gap-1 mt-2" role="img" aria-label={rating ? `${rating} out of 5 stars` : 'No rating'}>
        {[1, 2, 3, 4, 5].map((star) => (
          <span key={star} className={`text-2xl ${star <= rating ? 'text-accent drop-shadow-sm' : 'text-secondary/15'}`}>
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
              className="mushroom-tag bg-primary/10 text-primary border border-primary/15"
            >
              {item.replace(/_/g, ' ')}
            </span>
          ))}
        </div>
      );
    } catch {
      return <span className="text-sm">{value}</span>;
    }
  };

  const renderField = (label: string, value: string | number | undefined) => {
    if (!value) return null;
    return (
      <div className="bg-background/50 rounded-lg p-3 border border-secondary/5">
        <span className="text-xs uppercase tracking-wider text-text-light/70 font-medium">{label}</span>
        <p className="font-medium text-text mt-0.5">{value}</p>
      </div>
    );
  };

  return (
    <div className="min-h-screen bg-background">
      <header className="bg-surface/90 backdrop-blur-sm shadow-md border-b border-secondary/10">
        <div className="max-w-7xl mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <Link to="/books" className="text-primary hover:text-text-light transition-colors font-medium flex items-center gap-2" aria-label="Back to books list">
              <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <path d="M19 12H5M12 19l-7-7 7-7" />
              </svg>
              Back to Books
            </Link>
            {book && isAdmin && (
              <div className="flex gap-3">
                <button
                  onClick={() => setShowEditModal(true)}
                  aria-label="Edit book"
                  className="px-4 py-2 border border-secondary/30 rounded-xl text-text hover:bg-background focus:outline-none focus:ring-2 focus:ring-primary/50 focus:ring-offset-2 transition-all duration-200 text-sm font-medium flex items-center gap-1.5"
                >
                  <svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                  </svg>
                  Edit
                </button>
                <button
                  onClick={() => setShowDeleteModal(true)}
                  aria-label="Delete book"
                  className="px-4 py-2 border border-error/30 rounded-xl text-error hover:bg-error/5 focus:outline-none focus:ring-2 focus:ring-error/50 focus:ring-offset-2 transition-all duration-200 text-sm font-medium flex items-center gap-1.5"
                >
                  <svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <polyline points="3 6 5 6 21 6" />
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                  </svg>
                  Delete
                </button>
              </div>
            )}
          </div>
        </div>
      </header>

      <main className="max-w-5xl mx-auto px-4 py-8">
        {loading ? (
          <div className="text-center py-16 animate-gentle-pulse">
            <svg className="w-12 h-12 mx-auto text-primary/30 animate-spin" viewBox="0 0 40 40" fill="none" aria-hidden="true">
              <circle cx="20" cy="20" r="18" stroke="currentColor" strokeWidth="2" strokeDasharray="4 4" />
              <circle cx="20" cy="20" r="12" stroke="currentColor" strokeWidth="1.5" strokeDasharray="3 3" />
              <circle cx="20" cy="20" r="6" stroke="currentColor" strokeWidth="1" strokeDasharray="2 2" />
            </svg>
            <p className="text-text-light mt-4 font-accent text-xl" aria-live="polite">Opening the book...</p>
          </div>
        ) : error ? (
          <div className="text-center py-12" role="alert">
            <p className="text-error">{error}</p>
          </div>
        ) : book ? (
          <div className="bg-surface rounded-2xl shadow-lg border border-secondary/5 overflow-hidden animate-fade-in-up">
            {/* Cover image */}
            {book.cover_image_url ? (
              <div className="relative">
                <img
                  src={book.cover_image_url}
                  alt={book.title}
                  className="w-full h-64 md:h-80 object-cover"
                />
                <div className="absolute inset-0 bg-gradient-to-t from-surface via-transparent to-transparent" />
                {book.isbn && (
                  <a
                    href={`https://openlibrary.org/isbn/${book.isbn}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="absolute bottom-2 right-3 text-[11px] text-white/60 hover:text-white/90 transition-colors z-10 underline underline-offset-2"
                  >
                    Cover via Open Library
                  </a>
                )}
              </div>
            ) : (
              <div className="w-full h-40 bg-gradient-to-br from-primary/10 via-secondary/5 to-accent/10 flex items-center justify-center">
                <svg className="w-20 h-20 text-primary/15" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                  <path d="M21 5c-1.11-.35-2.33-.5-3.5-.5-1.95 0-4.05.4-5.5 1.5-1.45-1.1-3.55-1.5-5.5-1.5S2.45 4.9 1 6v14.65c0 .25.25.5.5.5.1 0 .15-.05.25-.05C3.1 20.45 5.05 20 6.5 20c1.95 0 4.05.4 5.5 1.5 1.35-.85 3.8-1.5 5.5-1.5 1.65 0 3.35.3 4.75 1.05.1.05.15.05.25.05.25 0 .5-.25.5-.5V6c-.6-.45-1.25-.75-2-1zm0 13.5c-1.1-.35-2.3-.5-3.5-.5-1.7 0-4.15.65-5.5 1.5V8c1.35-.85 3.8-1.5 5.5-1.5 1.2 0 2.4.15 3.5.5v11.5z" />
                </svg>
              </div>
            )}

            <div className="p-6 md:p-8">
              {/* Title & author */}
              <h1 className="text-3xl md:text-4xl font-heading text-primary leading-tight">{book.title}</h1>
              {book.subtitle && (
                <p className="text-text-light mt-1 text-lg italic">{book.subtitle}</p>
              )}
              {book.authors && (
                <p className="text-text-light mt-2">by <span className="text-text font-medium">{book.authors}</span></p>
              )}
              {book.illustrators && (
                <p className="text-text-light mt-1">Illustrated by <span className="text-text font-medium">{book.illustrators}</span></p>
              )}

              {book.child_rating != null && renderStars(book.child_rating)}

              <VineDivider />

              {/* Basic metadata grid */}
              <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
                {renderField('Type', book.book_type)}
                {renderField('Pages', book.page_count)}
                {renderField('Publisher', book.publisher)}
                {renderField('Year', book.publication_year)}
                {renderField('ISBN', book.isbn)}
                {renderField('Condition', book.condition)}
                {renderField('Location', book.location)}
                {renderField('Read Count', book.read_count)}
              </div>

              <VineDivider />

              {/* Reading levels */}
              {book.reading_levels && (
                <div>
                  <span className="text-sm font-medium text-text-light uppercase tracking-wider">Reading Levels</span>
                  {renderTags(book.reading_levels)}
                </div>
              )}

              {/* Genres */}
              {book.genres && (
                <div className="mt-4">
                  <span className="text-sm font-medium text-text-light uppercase tracking-wider">Genres</span>
                  {renderTags(book.genres)}
                </div>
              )}

              {/* Themes */}
              {book.themes && (
                <div className="mt-4">
                  <span className="text-sm font-medium text-text-light uppercase tracking-wider">Themes</span>
                  {renderTags(book.themes)}
                </div>
              )}

              {/* Awards */}
              {book.awards && (
                <div className="mt-4">
                  <span className="text-sm font-medium text-text-light uppercase tracking-wider">Awards</span>
                  {renderTags(book.awards)}
                </div>
              )}

              {/* Gift info */}
              {(book.gift_from || book.gift_relationship || book.date_received) && (
                <>
                  <VineDivider />
                  <div>
                    <h3 className="font-heading text-lg text-primary mb-3 flex items-center gap-2">
                      <svg className="w-5 h-5 text-accent" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                        <path d="M20 6h-2.18c.11-.31.18-.65.18-1a3 3 0 0 0-3-3c-1.1 0-2.04.6-2.57 1.5h-.03c-.53-.9-1.47-1.5-2.57-1.5a3 3 0 0 0-3 3c0 .35.07.69.18 1H4a2 2 0 0 0-2 2v2a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2zm-5-1c0 .55-.45 1-1 1s-1-.45-1-1 .45-1 1-1 1 .45 1 1zm-8 0c0 .55-.45 1-1 1s-1-.45-1-1 .45-1 1-1 1 .45 1 1z" />
                      </svg>
                      Gift Information
                    </h3>
                    <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
                      {renderField('From', book.gift_from)}
                      {renderField('Relationship', book.gift_relationship)}
                      {renderField('Date Received', book.date_received)}
                    </div>
                  </div>
                </>
              )}

              {/* Notes */}
              {book.notes && (
                <>
                  <VineDivider />
                  <div>
                    <h3 className="font-heading text-lg text-primary mb-2 flex items-center gap-2">
                      <svg className="w-5 h-5 text-accent" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
                        <polyline points="14 2 14 8 20 8" />
                        <line x1="16" y1="13" x2="8" y2="13" />
                        <line x1="16" y1="17" x2="8" y2="17" />
                      </svg>
                      Notes
                    </h3>
                    <p className="text-text whitespace-pre-wrap bg-background/50 rounded-lg p-4 border border-secondary/5">{book.notes}</p>
                  </div>
                </>
              )}

              {/* Timestamps */}
              <VineDivider />
              <div className="text-xs text-text-light/60 flex items-center gap-4">
                <span>📖 Added: {new Date(book.created_at).toLocaleDateString()}</span>
                {book.updated_at !== book.created_at && (
                  <span>✏️ Updated: {new Date(book.updated_at).toLocaleDateString()}</span>
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
            aria-label="Cancel delete"
            className="px-4 py-2 border border-secondary/30 rounded-xl text-text hover:bg-background focus:outline-none focus:ring-2 focus:ring-primary/50 focus:ring-offset-2 transition-colors disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            onClick={handleDeleteBook}
            disabled={deleting}
            aria-label="Confirm delete book"
            className="px-4 py-2 bg-error text-white rounded-xl hover:bg-error/90 focus:outline-none focus:ring-2 focus:ring-error/50 focus:ring-offset-2 transition-colors font-medium disabled:opacity-50"
          >
            {deleting ? 'Deleting...' : 'Delete Book'}
          </button>
        </div>
      </Modal>
    </div>
  );
}
