export interface Book {
  id: number;
  isbn?: string;
  title: string;
  subtitle?: string;
  authors?: string;  // JSON array string
  illustrators?: string;
  publisher?: string;
  publication_year?: number;
  page_count?: number;
  book_type?: string;
  reading_levels?: string;  // JSON array string
  genres?: string;
  themes?: string;
  awards?: string;
  gift_from?: string;
  gift_relationship?: string;
  date_received?: string;
  condition?: string;
  location?: string;
  notes?: string;
  child_rating?: number;
  read_count: number;
  last_read_date?: string;
  cover_image_url?: string;
  cover_source?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateBookRequest {
  isbn?: string;
  title: string;
  subtitle?: string;
  authors?: string;
  illustrators?: string;
  publisher?: string;
  publication_year?: number;
  page_count?: number;
  book_type?: string;
  reading_levels?: string;
  genres?: string;
  themes?: string;
  awards?: string;
  gift_from?: string;
  gift_relationship?: string;
  date_received?: string;
  condition?: string;
  location?: string;
  notes?: string;
  child_rating?: number;
}

export interface UpdateBookRequest {
  isbn?: string;
  title?: string;
  subtitle?: string;
  authors?: string;
  illustrators?: string;
  publisher?: string;
  publication_year?: number;
  page_count?: number;
  book_type?: string;
  reading_levels?: string;
  genres?: string;
  themes?: string;
  awards?: string;
  gift_from?: string;
  gift_relationship?: string;
  date_received?: string;
  condition?: string;
  location?: string;
  notes?: string;
  child_rating?: number;
  read_count?: number;
  last_read_date?: string;
  cover_image_url?: string;
  cover_source?: string;
}

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

export interface APIResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
}
