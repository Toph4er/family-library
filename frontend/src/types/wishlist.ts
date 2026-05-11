export interface WishlistItem {
  id: number;
  title: string;
  author?: string;
  isbn?: string;
  reason?: string;
  priority: number;
  amazon_url?: string;
  thriftbooks_url?: string;
  other_urls?: string;
  cover_image_url?: string;
  requested_by?: string;
  requested_at: string;
  fulfilled: boolean;
  fulfilled_at?: string;
  notes?: string;
}

export interface CreateWishlistItemRequest {
  title: string;
  author?: string;
  isbn?: string;
  reason?: string;
  priority?: number;
  amazon_url?: string;
  thriftbooks_url?: string;
  other_urls?: string;
  notes?: string;
}
