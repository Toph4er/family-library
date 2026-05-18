const API_BASE = '/api/v1';

// Module-level CSRF token state.
let csrfToken: string | null = null;
let isRefreshingCSRF = false;
let pendingCSRFRequests: Array<{
  resolve: (token: string) => void;
  reject: (error: Error) => void;
}> = [];

// isSafeMethod returns true for HTTP methods that do not cause state changes
// and therefore do not require a CSRF token.
function isSafeMethod(method?: string): boolean {
  const m = (method || 'GET').toUpperCase();
  return m === 'GET' || m === 'HEAD' || m === 'OPTIONS';
}

// fetchCSRFToken obtains a fresh CSRF token from the server.
// GET /csrf is exempt from CSRF validation, so this call never needs a token.
async function fetchCSRFToken(): Promise<string> {
  const resp = await fetch(`${API_BASE}/csrf`, {
    credentials: 'include',
  });
  if (!resp.ok) {
    throw new Error(`Failed to fetch CSRF token: ${resp.status}`);
  }
  const data = await resp.json();
  return data.csrf_token as string;
}

// getCSRFToken returns the current token, fetching one if necessary.
// Concurrent calls are coalesced so only one network request is made.
export async function getCSRFToken(): Promise<string> {
  if (csrfToken) {
    return csrfToken;
  }

  if (isRefreshingCSRF) {
    // Wait for the in-flight refresh to complete.
    return new Promise<string>((resolve, reject) => {
      pendingCSRFRequests.push({ resolve, reject });
    });
  }

  isRefreshingCSRF = true;
  try {
    csrfToken = await fetchCSRFToken();
    // Resolve all waiters.
    pendingCSRFRequests.forEach(({ resolve }) => resolve(csrfToken!));
    pendingCSRFRequests = [];
    return csrfToken;
  } catch (err) {
    const error = err instanceof Error ? err : new Error(String(err));
    pendingCSRFRequests.forEach(({ reject }) => reject(error));
    pendingCSRFRequests = [];
    throw error;
  } finally {
    isRefreshingCSRF = false;
  }
}

// initCSRF fetches a CSRF token and stores it for subsequent requests.
// Call this once at app startup (e.g. in AuthContext) before making any
// unsafe API calls.
export async function initCSRF(): Promise<void> {
  await getCSRFToken();
}

// updateCSRFTokenFromResponse reads the X-CSRF-Token response header (set
// by the server after each successful token rotation) and updates the
// module-level copy.
function updateCSRFTokenFromResponse(headers: Headers): void {
  const newToken = headers.get('X-CSRF-Token');
  if (newToken) {
    csrfToken = newToken;
  }
}

// fetchAPI performs an authenticated API request with automatic CSRF token
// handling.
//
// For unsafe methods (POST/PUT/PATCH/DELETE) the X-CSRF-Token header is
// added automatically.  If the server responds with 403 (token mismatch),
// the token is refreshed and the request is retried once.
export async function fetchAPI(endpoint: string, options?: RequestInit): Promise<any> {
  const method = options?.method || 'GET';

  // Build headers from the base + caller-supplied values.
  const baseHeaders = new Headers({
    'Content-Type': 'application/json',
  });
  if (options?.headers) {
    const callerHeaders = options.headers instanceof Headers
      ? options.headers
      : new Headers(options.headers as Record<string, string>);
    for (const [key, value] of callerHeaders) {
      baseHeaders.set(key, value);
    }
  }

  // Attach CSRF token for unsafe methods.
  if (!isSafeMethod(method)) {
    const token = await getCSRFToken();
    baseHeaders.set('X-CSRF-Token', token);
  }

  // Build the fetch options without the caller's headers (already merged).
  const { headers: _callerHeaders, ...restOptions } = options || {};

  let response = await fetch(`${API_BASE}${endpoint}`, {
    headers: baseHeaders,
    credentials: 'include',
    ...restOptions,
  });

  // Update local token from the rotated value in the response header.
  updateCSRFTokenFromResponse(response.headers);

  // On 403 for an unsafe method, refresh the token and retry once.
  if (response.status === 403 && !isSafeMethod(method)) {
    csrfToken = null; // Invalidate current token.
    const newToken = await getCSRFToken();

    const retryHeaders = new Headers(baseHeaders);
    retryHeaders.set('X-CSRF-Token', newToken);

    response = await fetch(`${API_BASE}${endpoint}`, {
      headers: retryHeaders,
      credentials: 'include',
      ...restOptions,
    });

    updateCSRFTokenFromResponse(response.headers);
  }

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
  lookupISBN: (isbn: string, force?: boolean) => {
    const params = new URLSearchParams({ isbn: encodeURIComponent(isbn) });
    if (force) params.set('force', 'true');
    return fetchAPI(`/books/lookup-isbn?${params.toString()}`);
  },
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
