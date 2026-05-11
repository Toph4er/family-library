import { ReactNode } from 'react';
import { Navigate, useLocation } from 'react-router';

interface ProtectedRouteProps {
  children: ReactNode;
  adminOnly?: boolean;
}

export default function ProtectedRoute({ children, adminOnly = false }: ProtectedRouteProps) {
  // TODO: Implement auth check
  // For now, redirect to login
  const location = useLocation();

  // Placeholder - will be replaced with real auth check
  const isAuthenticated = false; // session check
  const isAdmin = false; // role check

  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  if (adminOnly && !isAdmin) {
    return <Navigate to="/" replace />;
  }

  return <>{children}</>;
}
