import { ReactNode } from 'react';
import { Navigate, useLocation } from 'react-router';

interface ProtectedRouteProps {
  children: ReactNode;
  adminOnly?: boolean;
}

export default function ProtectedRoute({ children, adminOnly = false }: ProtectedRouteProps) {
  // TODO: Implement real auth check via /api/v1/auth/me
  // For now, allow access to test the UI
  const isAuthenticated = true;
  const isAdmin = true;

  if (!isAuthenticated) {
    const location = useLocation();
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  if (adminOnly && !isAdmin) {
    return <Navigate to="/" replace />;
  }

  return <>{children}</>;
}
