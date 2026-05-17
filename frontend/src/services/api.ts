const API_BASE = '/api/v1';

// TODO: Add CSRF token handling
// TODO: Add session management

export async function fetchAPI(endpoint: string, options?: RequestInit): Promise<any> {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
    credentials: 'include',
    ...options,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Request failed' }));
    throw new Error(error.error || `HTTP ${response.status}`);
  }

  return response.json();
}

export const api = {
  // Auth
  login: (username: string, password: string) => fetchAPI('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  }),
  guestLogin: (password: string) => fetchAPI('/auth/guest-login', {
    method: 'POST',
    body: JSON.stringify({ password }),
  }),
  logout: () => fetchAPI('/auth/logout', { method: 'POST' }),
  me: () => fetchAPI('/auth/me'),

  // Books
  getBooks: (params?: Record<string, string>) => {
    const query = params ? '?' + new URLSearchParams(params).toString() : '';
    return fetchAPI(`/books${query}`);
  },
  getBook: (id: number) => fetchAPI(`/books/${id}`),
  createBook: (data: any) => fetchAPI('/books', {
    method: 'POST',
    body: JSON.stringify(data),
  }),
  updateBook: (id: number, data: any) => fetchAPI(`/books/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  }),
  deleteBook: (id: number) => fetchAPI(`/books/${id}`, {
    method: 'DELETE',
  }),
  importISBN: (isbn: string) => fetchAPI('/books/import-isbn', {
    method: 'POST',
    body: JSON.stringify({ isbn }),
  }),
  lookupISBN: (isbn: string) => fetchAPI(`/books/lookup-isbn?isbn=${encodeURIComponent(isbn)}`),
  getTags: (type: string) => fetchAPI(`/books/tags?type=${encodeURIComponent(type)}`),

  // Wishlist
  getWishlist: () => fetchAPI('/wishlist'),
  createWishlistItem: (data: any) => fetchAPI('/wishlist', {
    method: 'POST',
    body: JSON.stringify(data),
  }),
  updateWishlistItem: (id: number, data: any) => fetchAPI(`/wishlist/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  }),
  deleteWishlistItem: (id: number) => fetchAPI(`/wishlist/${id}`, {
    method: 'DELETE',
  }),
  fulfillWishlistItem: (id: number) => fetchAPI(`/wishlist/${id}/fulfill`, {
    method: 'PATCH',
  }),

  // Settings
  getSettings: () => fetchAPI('/settings'),
  updateSetting: (key: string, value: string) => fetchAPI(`/settings/${key}`, {
    method: 'PUT',
    body: JSON.stringify({ value }),
  }),

  // Admin
  getUsers: () => fetchAPI('/admin/users'),
  createUser: (data: any) => fetchAPI('/admin/users', {
    method: 'POST',
    body: JSON.stringify(data),
  }),
  updateUser: (id: number, data: any) => fetchAPI(`/admin/users/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  }),
  deleteUser: (id: number) => fetchAPI(`/admin/users/${id}`, {
    method: 'DELETE',
  }),
};
