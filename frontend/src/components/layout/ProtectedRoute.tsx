import { ReactNode } from 'react';
import { Navigate, useLocation } from 'react-router';
import { useAuth } from '../../context/AuthContext';

interface ProtectedRouteProps {
  children: ReactNode;
  adminOnly?: boolean;
}

export default function ProtectedRoute({ children, adminOnly = false }: ProtectedRouteProps) {
  const location = useLocation();
  const { user, isAdmin, loading } = useAuth();

  // Redirect to /books if we're still loading auth on the books route
  if (loading && location.pathname === '/books') {
    return null;
  }

  const isAuthenticated = user != null;

  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  if (adminOnly && !isAdmin) {
    return <Navigate to="/books" replace />;
  }

  return <>{children}</>;
}
