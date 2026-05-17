import { useState, useEffect } from 'react';
import { Link } from 'react-router';
import { api } from '../services/api';
import type { User } from '../types/user';
import Modal from '../components/ui/Modal';

interface UserFormData {
  username: string;
  password: string;
  display_name: string;
  role: string;
}

const emptyForm: UserFormData = {
  username: '',
  password: '',
  display_name: '',
  role: 'member',
};

export default function AdminPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // Create modal
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [createForm, setCreateForm] = useState<UserFormData>(emptyForm);
  const [creating, setCreating] = useState(false);

  // Edit modal
  const [showEditModal, setShowEditModal] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [editForm, setEditForm] = useState<UserFormData>(emptyForm);
  const [updating, setUpdating] = useState(false);

  // Delete confirmation
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deletingUser, setDeletingUser] = useState<User | null>(null);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    loadUsers();
  }, []);

  const loadUsers = async () => {
    try {
      setLoading(true);
      const data = await api.getUsers();
      setUsers(Array.isArray(data) ? data : []);
    } catch (err: any) {
      setError(err.message || 'Failed to load users');
    } finally {
      setLoading(false);
    }
  };

  // ── Create ──────────────────────────────────────────────

  const handleCreate = async () => {
    if (!createForm.username.trim()) return;
    try {
      setCreating(true);
      await api.createUser(createForm);
      setCreateForm(emptyForm);
      setShowCreateModal(false);
      await loadUsers();
    } catch (err: any) {
      setError(err.message || 'Failed to create user');
    } finally {
      setCreating(false);
    }
  };

  // ── Update ──────────────────────────────────────────────

  const openEditModal = (user: User) => {
    setEditingUser(user);
    setEditForm({
      username: user.username,
      password: '',
      display_name: user.display_name || '',
      role: user.role,
    });
    setShowEditModal(true);
  };

  const handleUpdate = async () => {
    if (!editingUser || !editForm.username.trim()) return;
    try {
      setUpdating(true);
      const payload = { ...editForm };
      if (!payload.password) {
        delete (payload as any).password;
      }
      await api.updateUser(editingUser.id, payload);
      setShowEditModal(false);
      setEditingUser(null);
      await loadUsers();
    } catch (err: any) {
      setError(err.message || 'Failed to update user');
    } finally {
      setUpdating(false);
    }
  };

  // ── Delete ──────────────────────────────────────────────

  const openDeleteModal = (user: User) => {
    setDeletingUser(user);
    setShowDeleteModal(true);
  };

  const handleDelete = async () => {
    if (!deletingUser) return;
    try {
      setDeleting(true);
      await api.deleteUser(deletingUser.id);
      setShowDeleteModal(false);
      setDeletingUser(null);
      await loadUsers();
    } catch (err: any) {
      setError(err.message || 'Failed to delete user');
    } finally {
      setDeleting(false);
    }
  };

  // ── Helpers ─────────────────────────────────────────────

  const formatDate = (dateStr: string) => {
    try {
      return new Date(dateStr).toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
      });
    } catch {
      return dateStr;
    }
  };

  const roleBadgeClass = (role: string) => {
    switch (role) {
      case 'admin':
        return 'bg-accent/15 text-accent border-accent/30';
      case 'editor':
        return 'bg-primary/10 text-primary border-primary/20';
      default:
        return 'bg-secondary/10 text-text-light border-secondary/15';
    }
  };

  // ── Render ──────────────────────────────────────────────

  return (
    <div className="min-h-screen bg-background" role="main">
      {/* Header */}
      <header className="bg-surface/90 backdrop-blur-sm shadow-md border-b border-secondary/10">
        <div className="max-w-7xl mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <svg className="w-6 h-6 text-primary" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path d="M12 1L3 5v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V5l-9-4zm-2 16l-4-4 1.41-1.41L10 14.17l6.59-6.59L18 9l-8 8z" />
              </svg>
              <h1 className="text-2xl font-heading text-primary">Admin Dashboard</h1>
            </div>
            <nav className="flex gap-4">
              <Link to="/books" className="text-primary hover:text-text-light transition-colors font-medium text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 rounded" aria-label="Go to Books">Books</Link>
              <Link to="/wishlist" className="text-primary hover:text-text-light transition-colors font-medium text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 rounded" aria-label="Go to Wishlist">Wishlist</Link>
              <Link to="/settings" className="text-primary hover:text-text-light transition-colors font-medium text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 rounded" aria-label="Go to Settings">Settings</Link>
              <button
                onClick={async () => {
                  await api.logout();
                  window.location.href = '/';
                }}
                className="text-error hover:text-error/80 transition-colors font-medium text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 rounded"
                aria-label="Logout"
              >
                Logout
              </button>
            </nav>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 py-8">
        {/* Welcome banner */}
        <div className="mb-8 bg-gradient-to-r from-primary/10 via-secondary/5 to-accent/10 rounded-2xl p-6 border border-secondary/10 animate-fade-in-up">
          <h2 className="text-2xl font-heading text-primary mb-1">Welcome to the Library</h2>
          <p className="text-text-light font-accent text-xl">Manage your enchanted collection from here</p>
        </div>

        {/* ── User Management Section ─────────────────── */}
        <section className="animate-fade-in-up">
          <div className="flex items-center justify-between mb-6">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
                <svg className="w-5 h-5 text-primary" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                  <path d="M16 11c1.66 0 2.99-1.34 2.99-3S17.66 5 16 5c-1.66 0-3 1.34-3 3s1.34 3 3 3zm-8 0c1.66 0 2.99-1.34 2.99-3S9.66 5 8 5C6.34 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5z" />
                </svg>
              </div>
              <h2 className="text-xl font-heading text-primary">Library Members</h2>
            </div>
            <button
              onClick={() => {
                setCreateForm(emptyForm);
                setShowCreateModal(true);
              }}
              className="px-5 py-2.5 bg-primary text-white rounded-xl
                         shadow-md shadow-primary/20 hover:shadow-lg hover:shadow-primary/30
                         hover:-translate-y-0.5 active:translate-y-0
                         font-medium text-sm transition-all duration-200 ease-in-out
                         flex items-center gap-2
                         focus:outline-none focus:ring-2 focus:ring-primary/50"
              aria-label="Add new member"
            >
              <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" aria-hidden="true">
                <path d="M12 5v14M5 12h14" />
              </svg>
              Add Member
            </button>
          </div>

          {/* Error toast */}
          {error && (
            <div className="mb-4 px-4 py-3 bg-error/10 border border-error/20 rounded-xl text-error text-sm animate-fade-in" role="alert">
              {error}
              <button onClick={() => setError('')} className="ml-2 underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-primary/50 rounded" aria-label="Dismiss error">Dismiss</button>
            </div>
          )}

          {/* Users table */}
          <div className="bg-surface rounded-2xl shadow-md border border-secondary/5 overflow-hidden">
            {loading ? (
              <div className="text-center py-16 animate-gentle-pulse">
                <svg className="w-12 h-12 mx-auto text-primary/30 animate-spin" viewBox="0 0 40 40" fill="none" aria-hidden="true">
                  <circle cx="20" cy="20" r="18" stroke="currentColor" strokeWidth="2" strokeDasharray="4 4" />
                  <circle cx="20" cy="20" r="12" stroke="currentColor" strokeWidth="1.5" strokeDasharray="3 3" />
                  <circle cx="20" cy="20" r="6" stroke="currentColor" strokeWidth="1" strokeDasharray="2 2" />
                </svg>
                <p className="text-text-light mt-4 font-accent text-xl">Gathering the members...</p>
              </div>
            ) : users.length === 0 ? (
              <div className="text-center py-16 animate-fade-in">
                <svg className="w-16 h-16 mx-auto text-primary/20 mb-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                  <path d="M16 11c1.66 0 2.99-1.34 2.99-3S17.66 5 16 5c-1.66 0-3 1.34-3 3s1.34 3 3 3zm-8 0c1.66 0 2.99-1.34 2.99-3S9.66 5 8 5C6.34 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5z" />
                </svg>
                <p className="text-text-light mb-2 font-accent text-2xl">No members in the library yet.</p>
                <p className="text-sm text-text-light/70">Add your first member to get started!</p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-secondary/10 bg-background/50">
                      <th className="text-left px-6 py-3.5 text-xs font-semibold uppercase tracking-wider text-text-light/70">Username</th>
                      <th className="text-left px-6 py-3.5 text-xs font-semibold uppercase tracking-wider text-text-light/70">Display Name</th>
                      <th className="text-left px-6 py-3.5 text-xs font-semibold uppercase tracking-wider text-text-light/70">Role</th>
                      <th className="text-left px-6 py-3.5 text-xs font-semibold uppercase tracking-wider text-text-light/70">Created</th>
                      <th className="text-right px-6 py-3.5 text-xs font-semibold uppercase tracking-wider text-text-light/70">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-secondary/5">
                    {users.map((user) => (
                      <tr key={user.id} className="hover:bg-background/50 transition-colors">
                        <td className="px-6 py-4">
                          <span className="font-medium text-primary">{user.username}</span>
                        </td>
                        <td className="px-6 py-4">
                          <span className="text-text-light">{user.display_name || '—'}</span>
                        </td>
                        <td className="px-6 py-4">
                          <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium border ${roleBadgeClass(user.role)}`}>
                            {user.role}
                          </span>
                        </td>
                        <td className="px-6 py-4 text-sm text-text-light/70">
                          {formatDate(user.created_at)}
                        </td>
                        <td className="px-6 py-4">
                          <div className="flex items-center justify-end gap-2">
                            <button
                              onClick={() => openEditModal(user)}
                              className="p-1.5 rounded-lg text-text-light/50 hover:text-primary hover:bg-primary/10 transition-colors focus:outline-none focus:ring-2 focus:ring-primary/50"
                              aria-label="Edit user"
                            >
                              <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                                <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                              </svg>
                            </button>
                            <button
                              onClick={() => openDeleteModal(user)}
                              className="p-1.5 rounded-lg text-text-light/50 hover:text-error hover:bg-error/10 transition-colors focus:outline-none focus:ring-2 focus:ring-primary/50"
                              aria-label="Delete user"
                            >
                              <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                                <polyline points="3 6 5 6 21 6" />
                                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                              </svg>
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </section>

        {/* ── Dashboard cards ─────────────────────────── */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mt-10 stagger-children">
          {/* Books card */}
          <Link
            to="/books"
            className="group bg-surface rounded-2xl shadow-md card-hover border border-secondary/5 overflow-hidden focus:outline-none focus:ring-2 focus:ring-primary/50"
            aria-label="View Books collection"
          >
            <div className="p-6">
              <div className="flex items-center gap-4 mb-4">
                <div className="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center group-hover:bg-primary/20 transition-colors">
                  <svg className="w-6 h-6 text-primary" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                    <path d="M21 5c-1.11-.35-2.33-.5-3.5-.5-1.95 0-4.05.4-5.5 1.5-1.45-1.1-3.55-1.5-5.5-1.5S2.45 4.9 1 6v14.65c0 .25.25.5.5.5.1 0 .15-.05.25-.05C3.1 20.45 5.05 20 6.5 20c1.95 0 4.05.4 5.5 1.5 1.35-.85 3.8-1.5 5.5-1.5 1.65 0 3.35.3 4.75 1.05.1.05.15.05.25.05.25 0 .5-.25.5-.5V6c-.6-.45-1.25-.75-2-1zm0 13.5c-1.1-.35-2.3-.5-3.5-.5-1.7 0-4.15.65-5.5 1.5V8c1.35-.85 3.8-1.5 5.5-1.5 1.2 0 2.4.15 3.5.5v11.5z" />
                  </svg>
                </div>
                <div>
                  <h2 className="font-heading text-xl text-primary group-hover:text-text transition-colors">Books</h2>
                  <p className="text-text-light text-sm">Manage your collection</p>
                </div>
              </div>
              <div className="flex items-center gap-1 text-primary group-hover:text-text-light transition-colors font-medium text-sm">
                View Collection
                <svg className="w-4 h-4 group-hover:translate-x-1 transition-transform" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <path d="M5 12h14M12 5l7 7-7 7" />
                </svg>
              </div>
            </div>
          </Link>

          {/* Wishlist card */}
          <Link
            to="/wishlist"
            className="group bg-surface rounded-2xl shadow-md card-hover border border-secondary/5 overflow-hidden focus:outline-none focus:ring-2 focus:ring-primary/50"
            aria-label="View Wishlist"
          >
            <div className="p-6">
              <div className="flex items-center gap-4 mb-4">
                <div className="w-12 h-12 rounded-xl bg-accent/20 flex items-center justify-center group-hover:bg-accent/30 transition-colors">
                  <svg className="w-6 h-6 text-accent" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                    <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
                  </svg>
                </div>
                <div>
                  <h2 className="font-heading text-xl text-primary group-hover:text-text transition-colors">Wishlist</h2>
                  <p className="text-text-light text-sm">Books you want to add</p>
                </div>
              </div>
              <div className="flex items-center gap-1 text-primary group-hover:text-text-light transition-colors font-medium text-sm">
                View Wishlist
                <svg className="w-4 h-4 group-hover:translate-x-1 transition-transform" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <path d="M5 12h14M12 5l7 7-7 7" />
                </svg>
              </div>
            </div>
          </Link>

          {/* Settings card */}
          <Link
            to="/settings"
            className="group bg-surface rounded-2xl shadow-md card-hover border border-secondary/5 overflow-hidden focus:outline-none focus:ring-2 focus:ring-primary/50"
            aria-label="View Settings"
          >
            <div className="p-6">
              <div className="flex items-center gap-4 mb-4">
                <div className="w-12 h-12 rounded-xl bg-secondary/10 flex items-center justify-center group-hover:bg-secondary/20 transition-colors">
                  <svg className="w-6 h-6 text-secondary" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <circle cx="12" cy="12" r="3" />
                    <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
                  </svg>
                </div>
                <div>
                  <h2 className="font-heading text-xl text-primary group-hover:text-text transition-colors">Settings</h2>
                  <p className="text-text-light text-sm">Configure your library</p>
                </div>
              </div>
              <div className="flex items-center gap-1 text-primary group-hover:text-text-light transition-colors font-medium text-sm">
                View Settings
                <svg className="w-4 h-4 group-hover:translate-x-1 transition-transform" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <path d="M5 12h14M12 5l7 7-7 7" />
                </svg>
              </div>
            </div>
          </Link>
        </div>
      </main>

      {/* ── Create User Modal ─────────────────────────── */}
      <Modal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        title="Add New Member"
        size="md"
      >
        <form
          onSubmit={(e) => { e.preventDefault(); handleCreate(); }}
          className="space-y-4"
        >
          <div>
            <label className="block text-sm font-medium text-text-light mb-1.5">Username</label>
            <input
              type="text"
              required
              value={createForm.username}
              onChange={(e) => setCreateForm({ ...createForm, username: e.target.value })}
              className="w-full px-4 py-2.5 bg-background border border-secondary/20 rounded-xl
                         text-text placeholder:text-text-light/40
                         focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary/50
                         transition-all duration-200"
              placeholder="e.g. fairy_godmother"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-text-light mb-1.5">Password</label>
            <input
              type="password"
              required
              value={createForm.password}
              onChange={(e) => setCreateForm({ ...createForm, password: e.target.value })}
              className="w-full px-4 py-2.5 bg-background border border-secondary/20 rounded-xl
                         text-text placeholder:text-text-light/40
                         focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary/50
                         transition-all duration-200"
              placeholder="Choose a strong password"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-text-light mb-1.5">Display Name</label>
            <input
              type="text"
              value={createForm.display_name}
              onChange={(e) => setCreateForm({ ...createForm, display_name: e.target.value })}
              className="w-full px-4 py-2.5 bg-background border border-secondary/20 rounded-xl
                         text-text placeholder:text-text-light/40
                         focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary/50
                         transition-all duration-200"
              placeholder="e.g. Fairy Godmother"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-text-light mb-1.5">Role</label>
            <select
              value={createForm.role}
              onChange={(e) => setCreateForm({ ...createForm, role: e.target.value })}
              className="w-full px-4 py-2.5 bg-background border border-secondary/20 rounded-xl
                         text-text
                         focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary/50
                         transition-all duration-200"
            >
              <option value="member">Member</option>
              <option value="editor">Editor</option>
              <option value="admin">Admin</option>
            </select>
          </div>
          <div className="flex justify-end gap-3 pt-4">
            <button
              type="button"
              onClick={() => setShowCreateModal(false)}
              className="px-5 py-2.5 text-text-light hover:text-primary
                         rounded-xl hover:bg-background font-medium text-sm transition-colors
                         focus:outline-none focus:ring-2 focus:ring-primary/50"
              aria-label="Cancel"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={creating || !createForm.username.trim()}
              className="px-5 py-2.5 bg-primary text-white rounded-xl
                         shadow-md shadow-primary/20 hover:shadow-lg hover:shadow-primary/30
                         font-medium text-sm transition-all duration-200
                         disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:shadow-md
                         focus:outline-none focus:ring-2 focus:ring-primary/50"
              aria-label="Add member"
            >
              {creating ? 'Adding...' : 'Add Member'}
            </button>
          </div>
        </form>
      </Modal>

      {/* ── Edit User Modal ───────────────────────────── */}
      <Modal
        isOpen={showEditModal}
        onClose={() => { setShowEditModal(false); setEditingUser(null); }}
        title="Edit Member"
        size="md"
      >
        <form
          onSubmit={(e) => { e.preventDefault(); handleUpdate(); }}
          className="space-y-4"
        >
          <div>
            <label className="block text-sm font-medium text-text-light mb-1.5">Username</label>
            <input
              type="text"
              required
              value={editForm.username}
              onChange={(e) => setEditForm({ ...editForm, username: e.target.value })}
              className="w-full px-4 py-2.5 bg-background border border-secondary/20 rounded-xl
                         text-text placeholder:text-text-light/40
                         focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary/50
                         transition-all duration-200"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-text-light mb-1.5">
              Password
              <span className="text-text-light/50 font-normal ml-1">(leave blank to keep current)</span>
            </label>
            <input
              type="password"
              value={editForm.password}
              onChange={(e) => setEditForm({ ...editForm, password: e.target.value })}
              className="w-full px-4 py-2.5 bg-background border border-secondary/20 rounded-xl
                         text-text placeholder:text-text-light/40
                         focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary/50
                         transition-all duration-200"
              placeholder="New password (optional)"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-text-light mb-1.5">Display Name</label>
            <input
              type="text"
              value={editForm.display_name}
              onChange={(e) => setEditForm({ ...editForm, display_name: e.target.value })}
              className="w-full px-4 py-2.5 bg-background border border-secondary/20 rounded-xl
                         text-text placeholder:text-text-light/40
                         focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary/50
                         transition-all duration-200"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-text-light mb-1.5">Role</label>
            <select
              value={editForm.role}
              onChange={(e) => setEditForm({ ...editForm, role: e.target.value })}
              className="w-full px-4 py-2.5 bg-background border border-secondary/20 rounded-xl
                         text-text
                         focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary/50
                         transition-all duration-200"
            >
              <option value="member">Member</option>
              <option value="editor">Editor</option>
              <option value="admin">Admin</option>
            </select>
          </div>
          <div className="flex justify-end gap-3 pt-4">
            <button
              type="button"
              onClick={() => { setShowEditModal(false); setEditingUser(null); }}
              className="px-5 py-2.5 text-text-light hover:text-primary
                         rounded-xl hover:bg-background font-medium text-sm transition-colors
                         focus:outline-none focus:ring-2 focus:ring-primary/50"
              aria-label="Cancel"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={updating || !editForm.username.trim()}
              className="px-5 py-2.5 bg-primary text-white rounded-xl
                         shadow-md shadow-primary/20 hover:shadow-lg hover:shadow-primary/30
                         font-medium text-sm transition-all duration-200
                         disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:shadow-md
                         focus:outline-none focus:ring-2 focus:ring-primary/50"
              aria-label="Save changes"
            >
              {updating ? 'Saving...' : 'Save Changes'}
            </button>
          </div>
        </form>
      </Modal>

      {/* ── Delete Confirmation Modal ─────────────────── */}
      <Modal
        isOpen={showDeleteModal}
        onClose={() => !deleting && setShowDeleteModal(false)}
        title="Remove Member"
        size="sm"
      >
        <div className="text-center">
          <div className="w-14 h-14 mx-auto mb-4 rounded-full bg-error/10 flex items-center justify-center">
            <svg className="w-7 h-7 text-error" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <polyline points="3 6 5 6 21 6" />
              <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
              <line x1="10" y1="11" x2="10" y2="17" />
              <line x1="14" y1="11" x2="14" y2="17" />
            </svg>
          </div>
          <p className="text-text mb-1">
            Are you sure you want to remove{' '}
            <strong className="text-primary">{deletingUser?.username}</strong>
            {' '}from the library?
          </p>
          <p className="text-sm text-text-light/60 mb-6">This action cannot be undone.</p>
          <div className="flex justify-center gap-3">
            <button
              onClick={() => setShowDeleteModal(false)}
              disabled={deleting}
              className="px-5 py-2.5 text-text-light hover:text-primary
                         rounded-xl hover:bg-background font-medium text-sm transition-colors
                         disabled:opacity-50
                         focus:outline-none focus:ring-2 focus:ring-primary/50"
              aria-label="Cancel"
            >
              Cancel
            </button>
            <button
              onClick={handleDelete}
              disabled={deleting}
              className="px-5 py-2.5 bg-error text-white rounded-xl
                         shadow-md shadow-error/20 hover:shadow-lg hover:shadow-error/30
                         font-medium text-sm transition-all duration-200
                         disabled:opacity-50 disabled:cursor-not-allowed
                         focus:outline-none focus:ring-2 focus:ring-primary/50"
              aria-label="Remove member"
            >
              {deleting ? 'Removing...' : 'Remove Member'}
            </button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
