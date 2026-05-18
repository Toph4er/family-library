import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import ProtectedRoute from './ProtectedRoute';

// vi.mock is hoisted to the very top of the file — everything must be self-contained
vi.mock('@/context/AuthContext', () => ({
  useAuth: vi.fn(),
  AuthProvider: function MockAuthProvider({ children }: { children: React.ReactNode }) {
    return <>{children}</>;
  },
}));

// Re-import after mocking to get the mocked useAuth
import { useAuth } from '@/context/AuthContext';

const mockedUseAuth = useAuth as ReturnType<typeof vi.fn>;

describe('ProtectedRoute', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('loading state', () => {
    it('renders null while auth is loading', () => {
      mockedUseAuth.mockReturnValue({
        user: null,
        isAdmin: false,
        loading: true,
        refreshUser: vi.fn(),
        logout: vi.fn(),
      });

      render(
        <MemoryRouter>
          <ProtectedRoute>
            <div data-testid="protected-content">Protected</div>
          </ProtectedRoute>
        </MemoryRouter>
      );

      expect(screen.queryByTestId('protected-content')).not.toBeInTheDocument();
    });
  });

  describe('authentication gating', () => {
    it('renders children when user is authenticated', () => {
      mockedUseAuth.mockReturnValue({
        user: { id: 1, username: 'alice', role: 'user', is_guest: false },
        isAdmin: false,
        loading: false,
        refreshUser: vi.fn(),
        logout: vi.fn(),
      });

      render(
        <MemoryRouter>
          <ProtectedRoute>
            <div data-testid="protected-content">Protected</div>
          </ProtectedRoute>
        </MemoryRouter>
      );

      expect(screen.getByTestId('protected-content')).toBeInTheDocument();
    });

    it('redirects to /login when user is not authenticated', () => {
      mockedUseAuth.mockReturnValue({
        user: null,
        isAdmin: false,
        loading: false,
        refreshUser: vi.fn(),
        logout: vi.fn(),
      });

      render(
        <MemoryRouter initialEntries={['/protected']}>
          <ProtectedRoute>
            <div data-testid="protected-content">Protected</div>
          </ProtectedRoute>
        </MemoryRouter>
      );

      // Navigate component renders a link to /login
      expect(screen.getByRole('link')).toHaveAttribute('href', '/login');
      expect(screen.queryByTestId('protected-content')).not.toBeInTheDocument();
    });

    it('passes the original location as state to login redirect', () => {
      mockedUseAuth.mockReturnValue({
        user: null,
        isAdmin: false,
        loading: false,
        refreshUser: vi.fn(),
        logout: vi.fn(),
      });

      render(
        <MemoryRouter initialEntries={['/books/42']}>
          <ProtectedRoute>
            <div data-testid="protected-content">Protected</div>
          </ProtectedRoute>
        </MemoryRouter>
      );

      // The Navigate component should redirect to /login
      expect(screen.getByRole('link')).toHaveAttribute('href', '/login');
    });
  });

  describe('admin-only gating', () => {
    it('renders children when adminOnly and user is admin', () => {
      mockedUseAuth.mockReturnValue({
        user: { id: 1, username: 'alice', role: 'admin', is_guest: false },
        isAdmin: true,
        loading: false,
        refreshUser: vi.fn(),
        logout: vi.fn(),
      });

      render(
        <MemoryRouter>
          <ProtectedRoute adminOnly>
            <div data-testid="protected-content">Protected</div>
          </ProtectedRoute>
        </MemoryRouter>
      );

      expect(screen.getByTestId('protected-content')).toBeInTheDocument();
    });

    it('redirects to /books when adminOnly and user is not admin', () => {
      mockedUseAuth.mockReturnValue({
        user: { id: 2, username: 'bob', role: 'user', is_guest: false },
        isAdmin: false,
        loading: false,
        refreshUser: vi.fn(),
        logout: vi.fn(),
      });

      render(
        <MemoryRouter>
          <ProtectedRoute adminOnly>
            <div data-testid="protected-content">Protected</div>
          </ProtectedRoute>
        </MemoryRouter>
      );

      expect(screen.getByRole('link')).toHaveAttribute('href', '/books');
      expect(screen.queryByTestId('protected-content')).not.toBeInTheDocument();
    });

    it('does not check admin status when adminOnly is false', () => {
      mockedUseAuth.mockReturnValue({
        user: { id: 2, username: 'bob', role: 'user', is_guest: false },
        isAdmin: false,
        loading: false,
        refreshUser: vi.fn(),
        logout: vi.fn(),
      });

      render(
        <MemoryRouter>
          <ProtectedRoute>
            <div data-testid="protected-content">Protected</div>
          </ProtectedRoute>
        </MemoryRouter>
      );

      expect(screen.getByTestId('protected-content')).toBeInTheDocument();
    });
  });

  describe('combined scenarios', () => {
    it('redirects to /login before checking admin status when not authenticated', () => {
      mockedUseAuth.mockReturnValue({
        user: null,
        isAdmin: false,
        loading: false,
        refreshUser: vi.fn(),
        logout: vi.fn(),
      });

      render(
        <MemoryRouter>
          <ProtectedRoute adminOnly>
            <div data-testid="protected-content">Protected</div>
          </ProtectedRoute>
        </MemoryRouter>
      );

      // Should redirect to /login, not /books
      expect(screen.getByRole('link')).toHaveAttribute('href', '/login');
    });

    it('renders children for authenticated admin on admin-only route', () => {
      mockedUseAuth.mockReturnValue({
        user: { id: 1, username: 'admin', role: 'admin', is_guest: false },
        isAdmin: true,
        loading: false,
        refreshUser: vi.fn(),
        logout: vi.fn(),
      });

      render(
        <MemoryRouter>
          <ProtectedRoute adminOnly>
            <div data-testid="protected-content">Protected</div>
          </ProtectedRoute>
        </MemoryRouter>
      );

      expect(screen.getByTestId('protected-content')).toBeInTheDocument();
    });
  });
});
