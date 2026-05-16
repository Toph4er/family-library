import { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router';
import { api } from '../services/api';

// ── Types ──────────────────────────────────────────────────────────────────

interface SettingsData {
  [key: string]: string;
}

interface GuestVisibility {
  [field: string]: boolean;
}

interface ToastMessage {
  id: number;
  type: 'success' | 'error';
  text: string;
}

// ── Field definitions ──────────────────────────────────────────────────────

interface SettingField {
  key: string;
  label: string;
  description: string;
  type: 'text' | 'select' | 'json-toggles';
  options?: { value: string; label: string }[];
}

const SETTING_FIELDS: SettingField[] = [
  {
    key: 'site_name',
    label: 'Site Name',
    description: 'Display name shown in the header and on the landing page.',
    type: 'text',
  },
  {
    key: 'site_tagline',
    label: 'Site Tagline',
    description: 'Short tagline displayed beneath the site name.',
    type: 'text',
  },
  {
    key: 'cover_image_provider',
    label: 'Cover Image Provider',
    description: 'Primary API used to fetch book cover images.',
    type: 'select',
    options: [
      { value: 'google_books', label: 'Google Books' },
      { value: 'open_library', label: 'Open Library' },
    ],
  },
  {
    key: 'cover_image_fallback',
    label: 'Cover Image Fallback',
    description: 'Fallback API when the primary provider fails.',
    type: 'select',
    options: [
      { value: 'google_books', label: 'Google Books' },
      { value: 'open_library', label: 'Open Library' },
    ],
  },
  {
    key: 'default_guest_visibility',
    label: 'Guest Visible Fields',
    description: 'Which book fields are visible to guest viewers.',
    type: 'json-toggles',
  },
];

const GUEST_VISIBILITY_FIELDS: { key: string; label: string; category: string }[] = [
  { key: 'title', label: 'Title', category: 'Basic' },
  { key: 'subtitle', label: 'Subtitle', category: 'Basic' },
  { key: 'authors', label: 'Authors', category: 'Basic' },
  { key: 'illustrators', label: 'Illustrators', category: 'Basic' },
  { key: 'isbn', label: 'ISBN', category: 'Basic' },
  { key: 'publisher', label: 'Publisher', category: 'Publication' },
  { key: 'publication_year', label: 'Publication Year', category: 'Publication' },
  { key: 'page_count', label: 'Page Count', category: 'Publication' },
  { key: 'book_type', label: 'Book Type', category: 'Classification' },
  { key: 'reading_levels', label: 'Reading Levels', category: 'Classification' },
  { key: 'genres', label: 'Genres', category: 'Classification' },
  { key: 'themes', label: 'Themes', category: 'Classification' },
  { key: 'awards', label: 'Awards', category: 'Classification' },
  { key: 'cover_image_url', label: 'Cover Image', category: 'Media' },
  { key: 'child_rating', label: 'Child Rating', category: 'Personal' },
  { key: 'read_count', label: 'Read Count', category: 'Personal' },
  { key: 'gift_from', label: 'Gift From', category: 'Personal' },
  { key: 'gift_relationship', label: 'Gift Relationship', category: 'Personal' },
  { key: 'condition', label: 'Condition', category: 'Private' },
  { key: 'location', label: 'Location', category: 'Private' },
  { key: 'notes', label: 'Notes', category: 'Private' },
  { key: 'date_received', label: 'Date Received', category: 'Private' },
  { key: 'last_read_date', label: 'Last Read Date', category: 'Private' },
];

// ── Toast component ────────────────────────────────────────────────────────

function Toast({ message, onDismiss }: { message: ToastMessage; onDismiss: (id: number) => void }) {
  useEffect(() => {
    const timer = setTimeout(() => onDismiss(message.id), 4000);
    return () => clearTimeout(timer);
  }, [message.id, onDismiss]);

  return (
    <div
      className={`flex items-center gap-3 px-5 py-3.5 rounded-xl shadow-lg border backdrop-blur-sm animate-fade-in ${
        message.type === 'success'
          ? 'bg-success/90 border-success/30 text-white'
          : 'bg-error/90 border-error/30 text-white'
      }`}
    >
      {message.type === 'success' ? (
        <svg className="w-5 h-5 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
          <path d="M20 6L9 17l-5-5" />
        </svg>
      ) : (
        <svg className="w-5 h-5 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
          <circle cx="12" cy="12" r="10" />
          <path d="M15 9l-6 6M9 9l6 6" />
        </svg>
      )}
      <span className="text-sm font-medium">{message.text}</span>
      <button
        onClick={() => onDismiss(message.id)}
        className="ml-2 opacity-70 hover:opacity-100 transition-opacity"
        aria-label="Dismiss notification"
      >
        <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <path d="M18 6L6 18M6 6l12 12" />
        </svg>
      </button>
    </div>
  );
}

// ── Main page ──────────────────────────────────────────────────────────────

export default function SettingsPage() {
  const [settings, setSettings] = useState<SettingsData>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState<Record<string, boolean>>({});
  const [drafts, setDrafts] = useState<SettingsData>({});
  const [guestVisibility, setGuestVisibility] = useState<GuestVisibility>({});
  const [toasts, setToasts] = useState<ToastMessage[]>([]);
  const [loadError, setLoadError] = useState('');

  let toastId = 0;
  const addToast = useCallback((type: 'success' | 'error', text: string) => {
    const id = ++toastId;
    setToasts((prev) => [...prev, { id, type, text }]);
  }, []);

  const dismissToast = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  // Load settings on mount
  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    try {
      setLoading(true);
      const response = await api.getSettings();
      // API returns { success: true, data: { key: value, ... } }
      const data = response.data || response;
      setSettings(data);
      setDrafts({ ...data });
      // Parse guest visibility JSON
      if (data.default_guest_visibility) {
        try {
          setGuestVisibility(JSON.parse(data.default_guest_visibility));
        } catch {
          setGuestVisibility({});
        }
      }
    } catch (err: any) {
      setLoadError(err.message || 'Failed to load settings');
    } finally {
      setLoading(false);
    }
  };

  // Save a single setting
  const saveSetting = async (key: string, value: string) => {
    try {
      setSaving((prev) => ({ ...prev, [key]: true }));
      await api.updateSetting(key, value);
      setSettings((prev) => ({ ...prev, [key]: value }));
      setDrafts((prev) => ({ ...prev, [key]: value }));
      addToast('success', `${SETTING_FIELDS.find((f) => f.key === key)?.label || key} saved`);
    } catch (err: any) {
      addToast('error', err.message || 'Failed to save setting');
    } finally {
      setSaving((prev) => ({ ...prev, [key]: false }));
    }
  };

  // Save guest visibility
  const saveGuestVisibility = async () => {
    const json = JSON.stringify(guestVisibility);
    await saveSetting('default_guest_visibility', json);
  };

  // Toggle a single guest visibility field
  const toggleGuestField = (field: string) => {
    setGuestVisibility((prev) => ({ ...prev, [field]: !prev[field] }));
  };

  // Check if guest visibility has unsaved changes
  const guestVisibilityDirty = (() => {
    const current = settings.default_guest_visibility || '{}';
    const draft = JSON.stringify(guestVisibility);
    return current !== draft;
  })();

  // ── Loading state ──────────────────────────────────────────────────────

  if (loading) {
    return (
      <div className="min-h-screen bg-background">
        <PageHeader />
        <main className="max-w-4xl mx-auto px-4 py-8">
          <div className="text-center py-20 animate-gentle-pulse">
            <svg className="w-16 h-16 mx-auto text-primary/30 animate-spin" viewBox="0 0 40 40" fill="none">
              <circle cx="20" cy="20" r="18" stroke="currentColor" strokeWidth="2" strokeDasharray="4 4" />
              <circle cx="20" cy="20" r="12" stroke="currentColor" strokeWidth="1.5" strokeDasharray="3 3" />
              <circle cx="20" cy="20" r="6" stroke="currentColor" strokeWidth="1" strokeDasharray="2 2" />
            </svg>
            <p className="text-text-light mt-6 font-accent text-2xl">Gathering the settings...</p>
          </div>
        </main>
      </div>
    );
  }

  // ── Error state ────────────────────────────────────────────────────────

  if (loadError) {
    return (
      <div className="min-h-screen bg-background">
        <PageHeader />
        <main className="max-w-4xl mx-auto px-4 py-8">
          <div className="bg-surface rounded-2xl shadow-lg border border-error/20 p-8 text-center animate-fade-in">
            <svg className="w-12 h-12 mx-auto text-error/60 mb-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="12" cy="12" r="10" />
              <path d="M12 8v4M12 16h.01" />
            </svg>
            <h2 className="text-xl font-heading text-primary mb-2">Unable to Load Settings</h2>
            <p className="text-error mb-6">{loadError}</p>
            <button
              onClick={loadSettings}
              className="px-6 py-2.5 bg-primary text-white rounded-xl hover:bg-primary/90 transition-colors font-medium text-sm"
            >
              Try Again
            </button>
          </div>
        </main>
      </div>
    );
  }

  // ── Group guest visibility fields by category ──────────────────────────

  const groupedVisibility = GUEST_VISIBILITY_FIELDS.reduce<Record<string, typeof GUEST_VISIBILITY_FIELDS>>((acc, field) => {
    if (!acc[field.category]) acc[field.category] = [];
    acc[field.category].push(field);
    return acc;
  }, {});

  // ── Render ─────────────────────────────────────────────────────────────

  return (
    <div className="min-h-screen bg-background">
      <PageHeader />

      <main className="max-w-4xl mx-auto px-4 py-8">
        {/* Page title with decorative vine */}
        <div className="mb-8 animate-fade-in-up">
          <h1 className="text-3xl font-heading text-primary mb-2">Site Settings</h1>
          <p className="text-text-light font-accent text-xl">Configure your woodland library</p>
          <div className="vine-divider">
            <svg className="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
              <path d="M17 8C8 10 5.9 16.17 3.82 21.34l1.89.66.95-2.3c.48.17.98.3 1.34.3C19 20 22 3 22 3c-1 2-8 2.25-13 3.25S2 11.5 2 13.5s1.75 3.75 1.75 3.75C5 16 17 8 17 8z" />
            </svg>
          </div>
        </div>

        {/* Settings cards */}
        <div className="space-y-6 stagger-children">
          {/* Simple text / select settings */}
          {SETTING_FIELDS.filter((f) => f.type !== 'json-toggles').map((field) => (
            <div
              key={field.key}
              className="bg-surface rounded-2xl shadow-lg border border-secondary/5 overflow-hidden card-hover"
            >
              <div className="px-6 py-5 border-b border-secondary/10">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <h2 className="font-heading text-lg text-primary flex items-center gap-2">
                      <FieldIcon type={field.type} />
                      {field.label}
                    </h2>
                    <p className="text-text-light text-sm mt-1">{field.description}</p>
                  </div>
                  <button
                    onClick={() => saveSetting(field.key, drafts[field.key] || '')}
                    disabled={saving[field.key]}
                    className={`flex-shrink-0 px-5 py-2 rounded-xl text-sm font-medium transition-all ${
                      saving[field.key]
                        ? 'bg-primary/30 text-white/60 cursor-wait'
                        : 'bg-primary text-white hover:bg-primary/90 hover:shadow-md active:scale-[0.98]'
                    }`}
                  >
                    {saving[field.key] ? (
                      <span className="flex items-center gap-2">
                        <svg className="w-4 h-4 animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
                          <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83" />
                        </svg>
                        Saving...
                      </span>
                    ) : (
                      'Save'
                    )}
                  </button>
                </div>
              </div>

              <div className="px-6 py-5">
                {field.type === 'text' && (
                  <input
                    type="text"
                    value={drafts[field.key] || ''}
                    onChange={(e) => setDrafts((prev) => ({ ...prev, [field.key]: e.target.value }))}
                    className="w-full px-4 py-3 rounded-xl border border-secondary/20 bg-background/50 text-text placeholder:text-text-light/50 focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-all"
                    placeholder={`Enter ${field.label.toLowerCase()}...`}
                  />
                )}

                {field.type === 'select' && field.options && (
                  <select
                    value={drafts[field.key] || ''}
                    onChange={(e) => setDrafts((prev) => ({ ...prev, [field.key]: e.target.value }))}
                    className="w-full px-4 py-3 rounded-xl border border-secondary/20 bg-background/50 text-text focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-all appearance-none cursor-pointer"
                    style={{
                      backgroundImage: `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath d='M2 4l4 4 4-4' stroke='%238b4513' stroke-width='1.5' fill='none'/%3E%3C/svg%3E")`,
                      backgroundRepeat: 'no-repeat',
                      backgroundPosition: 'right 1rem center',
                    }}
                  >
                    {field.options.map((opt) => (
                      <option key={opt.value} value={opt.value}>{opt.label}</option>
                    ))}
                  </select>
                )}
              </div>
            </div>
          ))}

          {/* Guest visibility toggles */}
          <div className="bg-surface rounded-2xl shadow-lg border border-secondary/5 overflow-hidden card-hover">
            <div className="px-6 py-5 border-b border-secondary/10">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <h2 className="font-heading text-lg text-primary flex items-center gap-2">
                    <FieldIcon type="json-toggles" />
                    Guest Visible Fields
                  </h2>
                  <p className="text-text-light text-sm mt-1">
                    Control which book fields are visible to guest viewers. Toggled-off fields are hidden.
                  </p>
                </div>
                <button
                  onClick={saveGuestVisibility}
                  disabled={saving['default_guest_visibility'] || !guestVisibilityDirty}
                  className={`flex-shrink-0 px-5 py-2 rounded-xl text-sm font-medium transition-all ${
                    saving['default_guest_visibility']
                      ? 'bg-primary/30 text-white/60 cursor-wait'
                      : guestVisibilityDirty
                        ? 'bg-primary text-white hover:bg-primary/90 hover:shadow-md active:scale-[0.98]'
                        : 'bg-primary/20 text-primary/40 cursor-not-allowed'
                  }`}
                >
                  {saving['default_guest_visibility'] ? (
                    <span className="flex items-center gap-2">
                      <svg className="w-4 h-4 animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
                        <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83" />
                      </svg>
                      Saving...
                    </span>
                  ) : (
                    'Save'
                  )}
                </button>
              </div>
            </div>

            <div className="px-6 py-5">
              {/* Quick actions */}
              <div className="flex gap-3 mb-5">
                <button
                  onClick={() => {
                    const allTrue: GuestVisibility = {};
                    GUEST_VISIBILITY_FIELDS.forEach((f) => { allTrue[f.key] = true; });
                    setGuestVisibility(allTrue);
                  }}
                  className="px-4 py-1.5 text-xs font-medium rounded-lg bg-success/10 text-success hover:bg-success/20 transition-colors"
                >
                  Show All
                </button>
                <button
                  onClick={() => {
                    const allFalse: GuestVisibility = {};
                    GUEST_VISIBILITY_FIELDS.forEach((f) => { allFalse[f.key] = false; });
                    setGuestVisibility(allFalse);
                  }}
                  className="px-4 py-1.5 text-xs font-medium rounded-lg bg-error/10 text-error hover:bg-error/20 transition-colors"
                >
                  Hide All
                </button>
                <button
                  onClick={() => {
                    if (settings.default_guest_visibility) {
                      try {
                        setGuestVisibility(JSON.parse(settings.default_guest_visibility));
                      } catch {
                        setGuestVisibility({});
                      }
                    }
                  }}
                  className="px-4 py-1.5 text-xs font-medium rounded-lg bg-secondary/10 text-secondary hover:bg-secondary/20 transition-colors"
                >
                  Reset to Saved
                </button>
              </div>

              {/* Grouped toggle grid */}
              {Object.entries(groupedVisibility).map(([category, fields]) => (
                <div key={category} className="mb-5 last:mb-0">
                  <h3 className="text-xs font-semibold uppercase tracking-wider text-text-light/60 mb-3">
                    {category}
                  </h3>
                  <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
                    {fields.map((field) => {
                      const checked = !!guestVisibility[field.key];
                      return (
                        <label
                          key={field.key}
                          className={`flex items-center gap-3 px-4 py-3 rounded-xl border cursor-pointer transition-all ${
                            checked
                              ? 'border-primary/30 bg-primary/5 hover:bg-primary/10'
                              : 'border-secondary/10 bg-background/30 hover:bg-background/60'
                          }`}
                        >
                          <div className="relative">
                            <input
                              type="checkbox"
                              checked={checked}
                              onChange={() => toggleGuestField(field.key)}
                              className="sr-only"
                            />
                            <div
                              className={`w-10 h-6 rounded-full transition-colors ${
                                checked ? 'bg-primary' : 'bg-secondary/20'
                              }`}
                            >
                              <div
                                className={`absolute top-0.5 left-0.5 w-5 h-5 rounded-full bg-white shadow-sm transition-transform ${
                                  checked ? 'translate-x-4' : 'translate-x-0'
                                }`}
                              />
                            </div>
                          </div>
                          <span className={`text-sm ${checked ? 'text-text font-medium' : 'text-text-light/70'}`}>
                            {field.label}
                          </span>
                        </label>
                      );
                    })}
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Note about guest password */}
          <div className="bg-surface rounded-2xl shadow-lg border border-secondary/5 overflow-hidden">
            <div className="px-6 py-5 border-b border-secondary/10">
              <h2 className="font-heading text-lg text-primary flex items-center gap-2">
                <svg className="w-5 h-5 text-warning" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
                </svg>
                Guest Password
              </h2>
            </div>
            <div className="px-6 py-5">
              <p className="text-text-light text-sm leading-relaxed">
                The guest password is managed through environment variables at deploy time and cannot be changed
                through this interface. To update the guest password, modify the <code className="px-1.5 py-0.5 bg-background rounded text-xs font-mono">GUEST_PASSWORD</code> variable in your deployment configuration.
              </p>
            </div>
          </div>
        </div>
      </main>

      {/* Toast notifications */}
      <div className="fixed bottom-6 right-6 z-50 flex flex-col gap-3">
        {toasts.map((toast) => (
          <Toast key={toast.id} message={toast} onDismiss={dismissToast} />
        ))}
      </div>
    </div>
  );
}

// ── Sub-components ─────────────────────────────────────────────────────────

function PageHeader() {
  return (
    <header className="bg-surface/90 backdrop-blur-sm shadow-md border-b border-secondary/10">
      <div className="max-w-7xl mx-auto px-4 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <svg className="w-6 h-6 text-secondary" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="12" cy="12" r="3" />
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
            </svg>
            <h1 className="text-2xl font-heading text-primary">Settings</h1>
          </div>
          <nav className="flex gap-4">
            <Link to="/books" className="text-primary hover:text-text-light transition-colors font-medium text-sm">Books</Link>
            <Link to="/wishlist" className="text-primary hover:text-text-light transition-colors font-medium text-sm">Wishlist</Link>
            <Link to="/admin" className="text-primary hover:text-text-light transition-colors font-medium text-sm">Admin</Link>
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
  );
}

function FieldIcon({ type }: { type: string }) {
  if (type === 'text') {
    return (
      <svg className="w-5 h-5 text-accent" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <path d="M4 7V4h16v3M9 20h6M12 4v16" />
      </svg>
    );
  }
  if (type === 'select') {
    return (
      <svg className="w-5 h-5 text-accent" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <rect x="3" y="3" width="18" height="18" rx="2" />
        <circle cx="8.5" cy="8.5" r="1.5" />
        <path d="M21 15l-5-5L5 21" />
      </svg>
    );
  }
  return (
    <svg className="w-5 h-5 text-accent" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
}
