import { useState, useEffect } from 'react';
import { Link } from 'react-router';
import { api } from '../services/api';
import { useAuth } from '../context/AuthContext';
import type { WishlistItem, CreateWishlistItemRequest } from '../types/wishlist';
import Modal from '../components/ui/Modal';

// Priority badge component
const PriorityBadge = ({ priority }: { priority: number }) => {
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

// Wishlist form for adding/editing items
interface WishlistFormProps {
  item?: WishlistItem | null;
  onSubmit: (data: CreateWishlistItemRequest) => void;
  onCancel: () => void;
}

function WishlistForm({ item, onSubmit, onCancel }: WishlistFormProps) {
  const isEdit = !!item;

  const [title, setTitle] = useState(item?.title || '');
  const [author, setAuthor] = useState(item?.author || '');
  const [isbn, setIsbn] = useState(item?.isbn || '');
  const [reason, setReason] = useState(item?.reason || '');
  const [priority, setPriority] = useState(item?.priority ?? 5);
  const [amazon_url, setAmazonUrl] = useState(item?.amazon_url || '');
  const [thriftbooks_url, setThriftbooksUrl] = useState(item?.thriftbooks_url || '');
  const [notes, setNotes] = useState(item?.notes || '');

  const [errors, setErrors] = useState<Record<string, string>>({});

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};
    if (!title.trim()) newErrors.title = 'Title is required';
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!validate()) return;

    const data: CreateWishlistItemRequest = {
      title: title.trim(),
      author: author.trim() || undefined,
      isbn: isbn.replace(/\D/g, '').trim() || undefined,
      reason: reason.trim() || undefined,
      priority,
      amazon_url: amazon_url.trim() || undefined,
      thriftbooks_url: thriftbooks_url.trim() || undefined,
      notes: notes.trim() || undefined,
    };

    onSubmit(data);
  };

  const inputClass =
    'w-full px-3 py-2 border border-secondary/30 rounded-md bg-surface text-text focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-transparent';
  const labelClass = 'block text-sm font-medium text-text-light mb-1';
  const errorClass = 'text-error text-xs mt-1';

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {/* Title */}
      <div>
        <label htmlFor="wishlist-title" className={labelClass}>
          Title <span className="text-error">*</span>
        </label>
        <input
          id="wishlist-title"
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          className={`${inputClass} ${errors.title ? 'border-error ring-1 ring-error' : ''}`}
          placeholder="Book title"
        />
        {errors.title && <p className={errorClass}>{errors.title}</p>}
      </div>

      {/* Author & ISBN */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label htmlFor="wishlist-author" className={labelClass}>Author</label>
          <input
            id="wishlist-author"
            type="text"
            value={author}
            onChange={(e) => setAuthor(e.target.value)}
            className={inputClass}
            placeholder="Author name"
          />
        </div>
        <div>
          <label htmlFor="wishlist-isbn" className={labelClass}>ISBN</label>
          <input
            id="wishlist-isbn"
            type="text"
            value={isbn}
            onChange={(e) => setIsbn(e.target.value.replace(/\D/g, ''))}
            className={inputClass}
            placeholder="978-..."
          />
        </div>
      </div>

      {/* Reason */}
      <div>
        <label htmlFor="wishlist-reason" className={labelClass}>Why do you want it?</label>
        <textarea
          id="wishlist-reason"
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          rows={2}
          className={inputClass}
          placeholder="Because it looks magical..."
        />
      </div>

      {/* Priority */}
      <div>
        <label htmlFor="wishlist-priority" className={labelClass}>Priority (1-10)</label>
        <div className="flex items-center gap-3">
          <input
            id="wishlist-priority"
            type="range"
            min={1}
            max={10}
            value={priority}
            onChange={(e) => setPriority(Number(e.target.value))}
            className="flex-1 accent-primary"
          />
          <span className="text-lg font-medium text-primary w-8 text-center">{priority}</span>
        </div>
      </div>

      {/* Purchase links */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label htmlFor="wishlist-amazon" className={labelClass}>Amazon Link</label>
          <input
            id="wishlist-amazon"
            type="url"
            value={amazon_url}
            onChange={(e) => setAmazonUrl(e.target.value)}
            className={inputClass}
            placeholder="https://amazon.com/..."
          />
        </div>
        <div>
          <label htmlFor="wishlist-thriftbooks" className={labelClass}>ThriftBooks Link</label>
          <input
            id="wishlist-thriftbooks"
            type="url"
            value={thriftbooks_url}
            onChange={(e) => setThriftbooksUrl(e.target.value)}
            className={inputClass}
            placeholder="https://thriftbooks.com/..."
          />
        </div>
      </div>

      {/* Notes */}
      <div>
        <label htmlFor="wishlist-notes" className={labelClass}>Notes</label>
        <textarea
          id="wishlist-notes"
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
          rows={2}
          className={inputClass}
          placeholder="Any additional notes..."
        />
      </div>

      {/* Actions */}
      <div className="flex justify-end gap-3 pt-4 border-t border-secondary/20">
        <button
          type="button"
          onClick={onCancel}
          className="px-4 py-2 border border-secondary/30 rounded-xl text-text hover:bg-background transition-colors"
        >
          Cancel
        </button>
        <button
          type="submit"
          className="px-6 py-2 bg-primary text-white rounded-xl hover:bg-primary/90 transition-colors font-medium shadow-md shadow-primary/20"
        >
          {isEdit ? 'Save Changes' : 'Add to Wishlist'}
        </button>
      </div>
    </form>
  );
}

export default function WishlistPage() {
  const { isAdmin, logout } = useAuth();
  const [items, setItems] = useState<WishlistItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // Modal state
  const [showAddModal, setShowAddModal] = useState(false);
  const [editingItem, setEditingItem] = useState<WishlistItem | null>(null);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [deleting, setDeleting] = useState(false);

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

  const handleCreateItem = async (data: CreateWishlistItemRequest) => {
    try {
      await api.createWishlistItem(data);
      setShowAddModal(false);
      await loadWishlist();
    } catch (err: any) {
      setError(err.message || 'Failed to add wishlist item');
    }
  };

  const handleUpdateItem = async (data: CreateWishlistItemRequest) => {
    if (!editingItem) return;
    try {
      await api.updateWishlistItem(editingItem.id, data);
      setEditingItem(null);
      await loadWishlist();
    } catch (err: any) {
      setError(err.message || 'Failed to update wishlist item');
    }
  };

  const handleDeleteItem = async () => {
    if (deletingId == null) return;
    try {
      setDeleting(true);
      await api.deleteWishlistItem(deletingId);
      setDeletingId(null);
      await loadWishlist();
    } catch (err: any) {
      setError(err.message || 'Failed to delete wishlist item');
      setDeleting(false);
    }
  };

  const handleFulfillItem = async (id: number) => {
    try {
      await api.fulfillWishlistItem(id);
      await loadWishlist();
    } catch (err: any) {
      setError(err.message || 'Failed to update wishlist item');
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
          <div className="flex items-center justify-between flex-wrap gap-4">
            <div className="flex items-center gap-3">
              {/* Star icon */}
              <svg className="w-6 h-6 text-accent" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
              </svg>
              <h1 className="text-2xl font-heading text-primary">Wishlist</h1>
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
                  Add to Wishlist
                </button>
              )}
              <nav className="flex gap-4">
                <Link to="/books" className="text-primary hover:text-text-light transition-colors font-medium text-sm">Books</Link>
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
                      {item.notes && (
                        <p className={`text-sm mt-1 italic ${item.fulfilled ? 'text-text-light/40' : 'text-text-light/70'}`}>
                          Note: {item.notes}
                        </p>
                      )}
                      <div className={`flex flex-wrap gap-x-4 gap-y-1 mt-2 text-xs ${item.fulfilled ? 'text-text-light/40' : 'text-text-light/60'}`}>
                        {item.requested_by && (
                          <span>Requested by {item.requested_by}</span>
                        )}
                        <span>Added {new Date(item.requested_at).toLocaleDateString()}</span>
                      </div>
                    </div>
                    <div className="flex-shrink-0 flex flex-col items-end gap-2">
                      <PriorityBadge priority={item.priority} />
                      {isAdmin && !item.fulfilled && (
                        <button
                          onClick={() => handleFulfillItem(item.id)}
                          className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-medium bg-success/10 text-success hover:bg-success/20 transition-colors border border-success/15"
                          title="Mark as fulfilled"
                        >
                          <svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                            <polyline points="20 6 9 17 4 12" />
                          </svg>
                          Fulfill
                        </button>
                      )}
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
                      Fulfilled on {new Date(item.fulfilled_at).toLocaleDateString()}
                    </p>
                  )}

                  {/* Admin action buttons */}
                  {isAdmin && (
                    <div className="mt-4 pt-3 border-t border-secondary/10 flex justify-end gap-2">
                      <button
                        onClick={() => setEditingItem(item)}
                        className="px-3 py-1.5 border border-secondary/30 rounded-lg text-text-light hover:bg-background transition-all duration-200 text-xs font-medium flex items-center gap-1.5"
                      >
                        <svg className="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                          <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                          <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                        </svg>
                        Edit
                      </button>
                      <button
                        onClick={() => setDeletingId(item.id)}
                        className="px-3 py-1.5 border border-error/30 rounded-lg text-error hover:bg-error/5 transition-all duration-200 text-xs font-medium flex items-center gap-1.5"
                      >
                        <svg className="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                          <polyline points="3 6 5 6 21 6" />
                          <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                        </svg>
                        Delete
                      </button>
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </main>

      {/* Add Item Modal */}
      <Modal
        isOpen={showAddModal}
        onClose={() => setShowAddModal(false)}
        title="Add to Wishlist"
        size="lg"
      >
        <WishlistForm
          onSubmit={handleCreateItem}
          onCancel={() => setShowAddModal(false)}
        />
      </Modal>

      {/* Edit Item Modal */}
      <Modal
        isOpen={!!editingItem}
        onClose={() => setEditingItem(null)}
        title="Edit Wishlist Item"
        size="lg"
      >
        {editingItem && (
          <WishlistForm
            item={editingItem}
            onSubmit={handleUpdateItem}
            onCancel={() => setEditingItem(null)}
          />
        )}
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={deletingId !== null}
        onClose={() => !deleting && setDeletingId(null)}
        title="Delete Wishlist Item"
        size="sm"
      >
        <p className="text-text mb-6">
          Are you sure you want to remove{' '}
          <strong className="text-primary">
            {items.find((i) => i.id === deletingId)?.title}
          </strong>{' '}
          from the wishlist? This action cannot be undone.
        </p>
        <div className="flex justify-end gap-3">
          <button
            onClick={() => setDeletingId(null)}
            disabled={deleting}
            className="px-4 py-2 border border-secondary/30 rounded-xl text-text hover:bg-background transition-colors disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            onClick={handleDeleteItem}
            disabled={deleting}
            className="px-4 py-2 bg-error text-white rounded-xl hover:bg-error/90 transition-colors font-medium disabled:opacity-50"
          >
            {deleting ? 'Deleting...' : 'Remove'}
          </button>
        </div>
      </Modal>
    </div>
  );
}
