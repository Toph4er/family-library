import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router';
import { api } from '../services/api';
import type { Book } from '../types/book';

export default function BookDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [book, setBook] = useState<Book | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

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

  return (
    <div className="min-h-screen bg-background">
      <header className="bg-surface shadow-sm">
        <div className="max-w-7xl mx-auto px-4 py-4">
          <Link to="/books" className="text-primary hover:underline">← Back to Books</Link>
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
              {book.authors && (
                <p className="text-text-light mt-2">by {book.authors}</p>
              )}
              <div className="mt-6 grid grid-cols-2 gap-4">
                {book.book_type && (
                  <div>
                    <span className="text-sm text-text-light">Type:</span>
                    <p className="font-medium">{book.book_type}</p>
                  </div>
                )}
                {book.page_count && (
                  <div>
                    <span className="text-sm text-text-light">Pages:</span>
                    <p className="font-medium">{book.page_count}</p>
                  </div>
                )}
                {book.publisher && (
                  <div>
                    <span className="text-sm text-text-light">Publisher:</span>
                    <p className="font-medium">{book.publisher}</p>
                  </div>
                )}
                {book.publication_year && (
                  <div>
                    <span className="text-sm text-text-light">Year:</span>
                    <p className="font-medium">{book.publication_year}</p>
                  </div>
                )}
              </div>
            </div>
          </div>
        ) : null}
      </main>
    </div>
  );
}
