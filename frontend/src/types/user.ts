export interface User {
  id: number;
  username: string;
  role: string;
  display_name?: string;
  created_at: string;
  updated_at: string;
}

export interface SessionUser {
  id: number;
  username: string;
  role: string;
  is_guest: boolean;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface GuestLoginRequest {
  password: string;
}
