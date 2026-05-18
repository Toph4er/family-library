import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { AuthProvider, useAuth } from './AuthContext';

// Mock the api module
vi.mock('../services/api', () => ({
  api: {
    me: vi.fn(),
    logout: vi.fn(),
  },
  initCSRF: vi.fn(),
}));

// Dynamic import to get the mocked version
const getMockedApi = async () => {
  const mod = await import('../services/api');
  return { api: mod.api, initCSRF: mod.initCSRF };
};

// Test component that consumes the auth context
function TestConsumer() {
  const { user, isAdmin, loading, logout } = useAuth();
  return (
    <div>
      <div data-testid="loading">{String(loading)}</div>
      <div data-testid="username">{user?.username ?? 'null'}</div>
      <div data-testid="role">{user?.role ?? 'null'}</div>
      <div data-testid="is-admin">{String(isAdmin)}</div>
      <button onClick={logout}>Logout</button>
    </div>
  );
}

describe('AuthContext', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('loading state', () => {
    it('shows loading=true while auth check is in progress', async () => {
      const { api, initCSRF } = await getMockedApi();

      // Delay initCSRF so we can observe the initial loading state
      let initCSRFResolve: () => void;
      (initCSRF as ReturnType<typeof vi.fn>).mockImplementation(
        () => new Promise((resolve) => { initCSRFResolve = resolve; })
      );
      (api.me as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { user: { id: 1, username: 'alice', role: 'admin', is_guest: false } },
      });

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      // Initially loading (initCSRF hasn't resolved yet, so refreshUser hasn't run)
      expect(screen.getByTestId('loading')).toHaveTextContent('true');
      expect(screen.getByTestId('username')).toHaveTextContent('null');

      // Resolve initCSRF to trigger the auth check
      initCSRFResolve!();

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('false');
        expect(screen.getByTestId('username')).toHaveTextContent('alice');
      });
    });

    it('sets loading=false after successful auth check', async () => {
      const { api, initCSRF } = await getMockedApi();

      (api.me as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { user: { id: 1, username: 'alice', role: 'admin', is_guest: false } },
      });
      (initCSRF as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('false');
      });
    });

    it('sets loading=false after failed auth check', async () => {
      const { api, initCSRF } = await getMockedApi();

      (api.me as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Unauthorized'));
      (initCSRF as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('false');
      });
    });
  });

  describe('user data', () => {
    it('populates user data after successful auth', async () => {
      const { api, initCSRF } = await getMockedApi();

      (api.me as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { user: { id: 1, username: 'alice', role: 'admin', is_guest: false } },
      });
      (initCSRF as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('username')).toHaveTextContent('alice');
        expect(screen.getByTestId('role')).toHaveTextContent('admin');
      });
    });

    it('sets user to null when auth check fails', async () => {
      const { api, initCSRF } = await getMockedApi();

      (api.me as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Not found'));
      (initCSRF as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('username')).toHaveTextContent('null');
        expect(screen.getByTestId('role')).toHaveTextContent('null');
      });
    });

    it('computes isAdmin=true when user role is admin', async () => {
      const { api, initCSRF } = await getMockedApi();

      (api.me as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { user: { id: 1, username: 'alice', role: 'admin', is_guest: false } },
      });
      (initCSRF as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('is-admin')).toHaveTextContent('true');
      });
    });

    it('computes isAdmin=false when user role is not admin', async () => {
      const { api, initCSRF } = await getMockedApi();

      (api.me as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { user: { id: 2, username: 'bob', role: 'user', is_guest: false } },
      });
      (initCSRF as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('is-admin')).toHaveTextContent('false');
      });
    });

    it('computes isAdmin=false when user is null', async () => {
      const { api, initCSRF } = await getMockedApi();

      (api.me as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Unauthorized'));
      (initCSRF as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('is-admin')).toHaveTextContent('false');
      });
    });
  });

  describe('logout', () => {
    it('calls api.logout and sets user to null', async () => {
      const { api, initCSRF } = await getMockedApi();

      (api.me as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { user: { id: 1, username: 'alice', role: 'admin', is_guest: false } },
      });
      (api.logout as ReturnType<typeof vi.fn>).mockResolvedValue({ success: true });
      (initCSRF as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('username')).toHaveTextContent('alice');
      });

      // Click logout
      const logoutButton = screen.getByRole('button', { name: 'Logout' });
      logoutButton.click();

      await waitFor(() => {
        expect(screen.getByTestId('username')).toHaveTextContent('null');
      });

      expect(api.logout).toHaveBeenCalled();
    });

    it('sets user to null even if api.logout fails', async () => {
      const { api, initCSRF } = await getMockedApi();

      (api.me as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { user: { id: 1, username: 'alice', role: 'admin', is_guest: false } },
      });
      (api.logout as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Network error'));
      (initCSRF as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('username')).toHaveTextContent('alice');
      });

      const logoutButton = screen.getByRole('button', { name: 'Logout' });
      logoutButton.click();

      await waitFor(() => {
        expect(screen.getByTestId('username')).toHaveTextContent('null');
      });
    });
  });

  describe('refreshUser', () => {
    it('refreshes user data when called', async () => {
      const { api, initCSRF } = await getMockedApi();

      // Initial auth
      (api.me as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { user: { id: 1, username: 'alice', role: 'user', is_guest: false } },
      });
      (initCSRF as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);

      // Component that calls refreshUser
      function RefreshConsumer() {
        const { user, refreshUser } = useAuth();
        return (
          <div>
            <div data-testid="username">{user?.username ?? 'null'}</div>
            <button onClick={() => refreshUser()}>Refresh</button>
          </div>
        );
      }

      render(
        <AuthProvider>
          <RefreshConsumer />
        </AuthProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('username')).toHaveTextContent('alice');
      });

      // Change the mock for the refresh call
      (api.me as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { user: { id: 1, username: 'alice-updated', role: 'admin', is_guest: false } },
      });

      const refreshButton = screen.getByRole('button', { name: 'Refresh' });
      refreshButton.click();

      await waitFor(() => {
        expect(screen.getByTestId('username')).toHaveTextContent('alice-updated');
      });
    });
  });

  describe('default context value', () => {
    it('provides sensible defaults when used outside AuthProvider', () => {
      function DefaultConsumer() {
        const { user, isAdmin, loading } = useAuth();
        return (
          <div>
            <div data-testid="user">{String(user)}</div>
            <div data-testid="is-admin">{String(isAdmin)}</div>
            <div data-testid="loading">{String(loading)}</div>
          </div>
        );
      }

      render(<DefaultConsumer />);

      expect(screen.getByTestId('user')).toHaveTextContent('null');
      expect(screen.getByTestId('is-admin')).toHaveTextContent('false');
      expect(screen.getByTestId('loading')).toHaveTextContent('true');
    });
  });
});
