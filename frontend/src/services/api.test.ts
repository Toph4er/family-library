import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { fetchAPI, getCSRFToken, initCSRF, api } from './api';

// Helper to create a mock Response
function createMockResponse(status: number, body: any, extraHeaders?: Record<string, string>) {
  const headers = new Headers({
    'Content-Type': 'application/json',
    ...extraHeaders,
  });
  return {
    ok: status >= 200 && status < 300,
    status,
    headers,
    json: async () => body,
  };
}

describe('api.ts', () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.resetModules();
    fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  describe('fetchAPI', () => {
    it('returns parsed JSON on successful GET request', async () => {
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { success: true, data: { id: 1, title: 'Test Book' } })
      );

      // Re-import after mock is set up to get fresh module state
      const { fetchAPI } = await import('./api');

      const result = await fetchAPI('/books');
      expect(result).toEqual({ success: true, data: { id: 1, title: 'Test Book' } });
      expect(fetchMock).toHaveBeenCalledWith('/api/v1/books', {
        headers: expect.any(Headers),
        credentials: 'include',
      });
    });

    it('sends POST request with JSON body', async () => {
      // First call: CSRF token fetch
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { csrf_token: 'test-csrf-token' })
      );
      // Second call: actual POST
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { success: true })
      );

      const { fetchAPI } = await import('./api');

      const result = await fetchAPI('/books', {
        method: 'POST',
        body: JSON.stringify({ title: 'New Book' }),
      });

      expect(result).toEqual({ success: true });
      expect(fetchMock).toHaveBeenCalledTimes(2);

      // Verify the second call includes the CSRF token
      const secondCall = fetchMock.mock.calls[1];
      const headers = secondCall[1].headers;
      expect(headers.get('X-CSRF-Token')).toBe('test-csrf-token');
    });

    it('throws an error on HTTP error responses', async () => {
      fetchMock.mockResolvedValueOnce(
        createMockResponse(404, { error: 'Book not found' })
      );

      const { fetchAPI } = await import('./api');

      await expect(fetchAPI('/books/999')).rejects.toThrow('Book not found');
    });

    it('throws HTTP status when error body has no error field', async () => {
      fetchMock.mockResolvedValueOnce(
        createMockResponse(500, { message: 'Internal error' })
      );

      const { fetchAPI } = await import('./api');

      await expect(fetchAPI('/books')).rejects.toThrow('HTTP 500');
    });

    it('throws fallback error when response body is not JSON', async () => {
      const nonJsonResponse = {
        ok: false,
        status: 502,
        headers: new Headers({ 'Content-Type': 'text/plain' }),
        json: async () => { throw new Error('Not JSON'); },
      };
      fetchMock.mockResolvedValueOnce(nonJsonResponse);

      const { fetchAPI } = await import('./api');

      // The .catch() in the source returns { error: 'Request failed' }
      await expect(fetchAPI('/books')).rejects.toThrow('Request failed');
    });

    it('does not attach CSRF token for safe methods (GET)', async () => {
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { books: [] })
      );

      const { fetchAPI } = await import('./api');

      await fetchAPI('/books');

      const call = fetchMock.mock.calls[0];
      const headers = call[1].headers;
      expect(headers.get('X-CSRF-Token')).toBeNull();
    });

    it('attaches CSRF token for unsafe methods (POST)', async () => {
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { csrf_token: 'abc123' })
      );
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { success: true })
      );

      const { fetchAPI } = await import('./api');

      await fetchAPI('/books', { method: 'POST', body: '{}' });

      const secondCall = fetchMock.mock.calls[1];
      const headers = secondCall[1].headers;
      expect(headers.get('X-CSRF-Token')).toBe('abc123');
    });

    it('merges caller-supplied headers with base headers', async () => {
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { success: true })
      );

      const { fetchAPI } = await import('./api');

      await fetchAPI('/books', {
        headers: { 'X-Custom-Header': 'custom-value' },
      });

      const call = fetchMock.mock.calls[0];
      const headers = call[1].headers;
      expect(headers.get('Content-Type')).toBe('application/json');
      expect(headers.get('X-Custom-Header')).toBe('custom-value');
    });

    it('merges caller-supplied Headers instance', async () => {
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { success: true })
      );

      const { fetchAPI } = await import('./api');

      await fetchAPI('/books', {
        headers: new Headers({ 'X-Custom-Header': 'from-headers' }),
      });

      const call = fetchMock.mock.calls[0];
      const headers = call[1].headers;
      expect(headers.get('X-Custom-Header')).toBe('from-headers');
    });
  });

  describe('CSRF token retry on 403', () => {
    it('refreshes CSRF token and retries on 403 for unsafe methods', async () => {
      // First: get initial CSRF token
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { csrf_token: 'old-token' })
      );
      // Second: POST returns 403
      fetchMock.mockResolvedValueOnce(
        createMockResponse(403, { error: 'CSRF token mismatch' })
      );
      // Third: fetch new CSRF token
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { csrf_token: 'new-token' })
      );
      // Fourth: retried POST succeeds
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { success: true })
      );

      const { fetchAPI } = await import('./api');

      const result = await fetchAPI('/books', { method: 'POST', body: '{}' });

      expect(result).toEqual({ success: true });
      expect(fetchMock).toHaveBeenCalledTimes(4);

      // Verify the retry used the new token
      const retryCall = fetchMock.mock.calls[3];
      const headers = retryCall[1].headers;
      expect(headers.get('X-CSRF-Token')).toBe('new-token');
    });

    it('does not retry on 403 for safe methods', async () => {
      fetchMock.mockResolvedValueOnce(
        createMockResponse(403, { error: 'Forbidden' })
      );

      const { fetchAPI } = await import('./api');

      await expect(fetchAPI('/books')).rejects.toThrow('Forbidden');
      expect(fetchMock).toHaveBeenCalledTimes(1);
    });

    it('throws if retried request also returns 403', async () => {
      // Initial CSRF fetch
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { csrf_token: 'old-token' })
      );
      // First POST returns 403
      fetchMock.mockResolvedValueOnce(
        createMockResponse(403, { error: 'CSRF token mismatch' })
      );
      // Fetch new token
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { csrf_token: 'new-token' })
      );
      // Retried POST also returns 403
      fetchMock.mockResolvedValueOnce(
        createMockResponse(403, { error: 'Still forbidden' })
      );

      const { fetchAPI } = await import('./api');

      await expect(fetchAPI('/books', { method: 'POST', body: '{}' })).rejects.toThrow('Still forbidden');
    });
  });

  describe('CSRF token handling', () => {
    it('getCSRFToken fetches and caches a token', async () => {
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { csrf_token: 'fetched-token' })
      );

      const { getCSRFToken } = await import('./api');

      const token = await getCSRFToken();
      expect(token).toBe('fetched-token');
      expect(fetchMock).toHaveBeenCalledWith('/api/v1/csrf', {
        credentials: 'include',
      });
    });

    it('getCSRFToken returns cached token without re-fetching', async () => {
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { csrf_token: 'cached-token' })
      );

      const { getCSRFToken } = await import('./api');

      const token1 = await getCSRFToken();
      const token2 = await getCSRFToken();

      expect(token1).toBe('cached-token');
      expect(token2).toBe('cached-token');
      expect(fetchMock).toHaveBeenCalledTimes(1);
    });

    it('getCSRFToken coalesces concurrent calls', async () => {
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { csrf_token: 'coalesced-token' })
      );

      const { getCSRFToken } = await import('./api');

      const [token1, token2] = await Promise.all([
        getCSRFToken(),
        getCSRFToken(),
      ]);

      expect(token1).toBe('coalesced-token');
      expect(token2).toBe('coalesced-token');
      expect(fetchMock).toHaveBeenCalledTimes(1);
    });

    it('getCSRFToken rejects concurrent calls when fetch fails', async () => {
      fetchMock.mockResolvedValueOnce(
        createMockResponse(500, { error: 'Server error' })
      );

      const { getCSRFToken } = await import('./api');

      const p1 = getCSRFToken();
      const p2 = getCSRFToken();

      await expect(p1).rejects.toThrow('Failed to fetch CSRF token: 500');
      await expect(p2).rejects.toThrow('Failed to fetch CSRF token: 500');
    });

    it('initCSRF fetches a CSRF token', async () => {
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { csrf_token: 'init-token' })
      );

      const { initCSRF } = await import('./api');

      await initCSRF();
      expect(fetchMock).toHaveBeenCalledWith('/api/v1/csrf', {
        credentials: 'include',
      });
    });

    it('updates CSRF token from X-CSRF-Token response header', async () => {
      // Initial token fetch
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { csrf_token: 'initial' })
      );
      // POST with rotated token in header
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { success: true }, { 'X-CSRF-Token': 'rotated-token' })
      );

      const { fetchAPI, getCSRFToken } = await import('./api');

      await fetchAPI('/books', { method: 'POST', body: '{}' });

      // Subsequent getCSRFToken should return the rotated token
      const token = await getCSRFToken();
      expect(token).toBe('rotated-token');
    });
  });

  describe('api convenience methods', () => {
    it('api.login sends POST with credentials', async () => {
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { csrf_token: 'token' })
      );
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { success: true })
      );

      const { api } = await import('./api');

      await api.login('alice', 'password123');

      expect(fetchMock).toHaveBeenCalledTimes(2);
      const loginCall = fetchMock.mock.calls[1];
      expect(loginCall[0]).toBe('/api/v1/auth/login');
      // method is spread into restOptions via the fetch call
      expect(loginCall[1]).toHaveProperty('method', 'POST');
    });

    it('api.getBooks sends query parameters', async () => {
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { items: [], total: 0 })
      );

      const { api } = await import('./api');

      await api.getBooks({ page: '1', per_page: '10' });

      expect(fetchMock).toHaveBeenCalledWith(
        '/api/v1/books?page=1&per_page=10',
        expect.objectContaining({ credentials: 'include' })
      );
    });

    it('api.getBook fetches a single book by id', async () => {
      fetchMock.mockResolvedValueOnce(
        createMockResponse(200, { data: { id: 1, title: 'Test' } })
      );

      const { api } = await import('./api');

      await api.getBook(42);

      expect(fetchMock).toHaveBeenCalledWith(
        '/api/v1/books/42',
        expect.objectContaining({ credentials: 'include' })
      );
    });
  });
});
