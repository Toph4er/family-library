import { useState } from 'react';
import type { Book, CreateBookRequest, UpdateBookRequest } from '@/types/book';
import { api } from '@/services/api';
import TagInput from '@/components/ui/TagInput';

interface BookFormProps {
  book?: Book | null;
  onSubmit: (data: CreateBookRequest | UpdateBookRequest) => void;
  onCancel: () => void;
}

const BOOK_TYPES = [
  { value: '', label: 'Select type...' },
  { value: 'hardback', label: 'Hardback' },
  { value: 'paperback', label: 'Paperback' },
  { value: 'board_book', label: 'Board Book' },
  { value: 'ebook', label: 'E-book' },
  { value: 'audiobook', label: 'Audiobook' },
  { value: 'other', label: 'Other' },
];

const CONDITIONS = [
  { value: '', label: 'Select condition...' },
  { value: 'new', label: 'New' },
  { value: 'like_new', label: 'Like New' },
  { value: 'good', label: 'Good' },
  { value: 'fair', label: 'Fair' },
  { value: 'poor', label: 'Poor' },
  { value: 'damaged', label: 'Damaged' },
];

function parseJsonArray(value: string | undefined): string[] {
  if (!value) return [];
  try {
    const parsed = JSON.parse(value);
    return Array.isArray(parsed) ? parsed.map(String) : [];
  } catch {
    // If it's not valid JSON, treat as comma-separated
    return value.split(',').map((s) => s.trim()).filter(Boolean);
  }
}

function toJsonArray(values: string[]): string {
  return values.length > 0 ? JSON.stringify(values) : '';
}

export default function BookForm({ book, onSubmit, onCancel }: BookFormProps) {
  const isEdit = !!book;

  // Basic fields
  const [title, setTitle] = useState(book?.title || '');
  const [subtitle, setSubtitle] = useState(book?.subtitle || '');
  const [authors, setAuthors] = useState(book?.authors || '');
  const [illustrators, setIllustrators] = useState(book?.illustrators || '');
  const [isbn, setIsbn] = useState(book?.isbn || '');
  const [publisher, setPublisher] = useState(book?.publisher || '');
  const [publication_year, setPublicationYear] = useState(book?.publication_year?.toString() || '');
  const [page_count, setPageCount] = useState(book?.page_count?.toString() || '');
  const [book_type, setBookType] = useState(book?.book_type || '');
  const [condition, setCondition] = useState(book?.condition || '');
  const [notes, setNotes] = useState(book?.notes || '');
  const [child_rating, setChildRating] = useState(book?.child_rating ?? 0);
  const [cover_image_url, setCoverImageUrl] = useState(book?.cover_image_url || '');

  // JSON array fields (stored as comma-separated in UI)
  const [readingLevels, setReadingLevels] = useState<string[]>(parseJsonArray(book?.reading_levels));
  const [genres, setGenres] = useState<string[]>(parseJsonArray(book?.genres));
  const [themes, setThemes] = useState<string[]>(parseJsonArray(book?.themes));
  const [awards, setAwards] = useState<string[]>(parseJsonArray(book?.awards));

  // Advanced fields
  const [gift_from, setGiftFrom] = useState(book?.gift_from || '');
  const [gift_relationship, setGiftRelationship] = useState(book?.gift_relationship || '');
  const [date_received, setDateReceived] = useState(book?.date_received || '');
  const [location, setLocation] = useState(book?.location || '');

  const [errors, setErrors] = useState<Record<string, string>>({});
  const [showAdvanced, setShowAdvanced] = useState(false);

  // ISBN lookup state
  const [lookupLoading, setLookupLoading] = useState(false);
  const [lookupError, setLookupError] = useState<string | null>(null);

  const handleLookupISBN = async (event?: React.MouseEvent) => {
    const digits = isbn.replace(/\D/g, '');
    if (digits.length < 10) {
      setLookupError('Please enter a valid ISBN-10 or ISBN-13');
      return;
    }
    const force = event?.shiftKey || false;
    setLookupLoading(true);
    setLookupError(null);
    try {
      const data = await api.lookupISBN(digits, force);

      // Auto-fill fields from lookup result
      if (data.title) setTitle(data.title);
      if (data.subtitle) setSubtitle(data.subtitle);
      if (data.publisher) setPublisher(data.publisher);
      if (data.publication_year) setPublicationYear(data.publication_year.toString());
      if (data.page_count) setPageCount(data.page_count.toString());
      if (data.cover_image_url) setCoverImageUrl(data.cover_image_url);

      // Parse JSON array fields from lookup and populate form state

      // Authors: backend returns JSON array string; convert to comma-separated for text input
      if (data.authors) {
        try {
          const parsed = JSON.parse(data.authors);
          if (Array.isArray(parsed)) setAuthors(parsed.map(String).join(', '));
          else setAuthors(String(data.authors));
        } catch {
          // Not JSON, use as-is
          setAuthors(String(data.authors));
        }
      }

      // Illustrators: backend returns JSON array string; convert to comma-separated for text input
      if (data.illustrators) {
        try {
          const parsed = JSON.parse(data.illustrators);
          if (Array.isArray(parsed)) setIllustrators(parsed.map(String).join(', '));
          else setIllustrators(String(data.illustrators));
        } catch {
          // Not JSON, use as-is
          setIllustrators(String(data.illustrators));
        }
      }

      // Genres: backend returns JSON array string; populate multi-select
      if (data.genres) {
        try {
          const parsed = JSON.parse(data.genres);
          if (Array.isArray(parsed)) setGenres(parsed.map(String));
        } catch {
          // Not JSON, skip
        }
      }
      if (data.reading_levels) {
        try {
          const parsed = JSON.parse(data.reading_levels);
          if (Array.isArray(parsed)) setReadingLevels(parsed.map(String));
        } catch {
          // Not JSON, skip
        }
      }
      if (data.themes) {
        try {
          const parsed = JSON.parse(data.themes);
          if (Array.isArray(parsed)) setThemes(parsed.map(String));
        } catch {
          // Not JSON, skip
        }
      }
      if (data.awards) {
        try {
          const parsed = JSON.parse(data.awards);
          if (Array.isArray(parsed)) setAwards(parsed.map(String));
        } catch {
          // Not JSON, skip
        }
      }
    } catch (err: any) {
      if (err.message?.includes('not found')) {
        setLookupError('No book found for this ISBN');
      } else if (err.message?.includes('book lookup service is unavailable')) {
        setLookupError('Book lookup service is unavailable. Please try again later.');
      } else {
        setLookupError(err.message || 'Failed to look up ISBN');
      }
    } finally {
      setLookupLoading(false);
    }
  };

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};
    if (!title.trim()) newErrors.title = 'Title is required';
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!validate()) return;

    const data: CreateBookRequest | UpdateBookRequest = {
      title: title.trim(),
      subtitle: subtitle.trim() || undefined,
      authors: authors.trim() || undefined,
      illustrators: illustrators.trim() || undefined,
      isbn: isbn.replace(/\D/g, '').trim() || undefined,
      publisher: publisher.trim() || undefined,
      publication_year: publication_year ? parseInt(publication_year, 10) : undefined,
      page_count: page_count ? parseInt(page_count, 10) : undefined,
      book_type: book_type || undefined,
      reading_levels: toJsonArray(readingLevels) || undefined,
      genres: toJsonArray(genres) || undefined,
      themes: toJsonArray(themes) || undefined,
      awards: toJsonArray(awards) || undefined,
      gift_from: gift_from.trim() || undefined,
      gift_relationship: gift_relationship.trim() || undefined,
      date_received: date_received || undefined,
      condition: condition || undefined,
      location: location.trim() || undefined,
      notes: notes.trim() || undefined,
      child_rating,
      cover_image_url: cover_image_url.trim() || undefined,
    };

    onSubmit(data);
  };

  const inputClass =
    'w-full px-3 py-2 border border-secondary/30 rounded-md bg-surface text-text focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-transparent';
  const labelClass = 'block text-sm font-medium text-text-light mb-1';
  const errorClass = 'text-error text-xs mt-1';

  // Build error summary for screen readers (WCAG 2.1 AA)
  const errorSummaryId = 'form-error-summary';
  const hasErrors = Object.keys(errors).length > 0;

  return (
    <form onSubmit={handleSubmit} className="space-y-4" noValidate>
      {/* Error summary announced to screen readers */}
      {hasErrors && (
        <div
          id={errorSummaryId}
          role="alert"
          className="p-3 bg-error/10 border border-error/20 rounded-xl text-error text-sm"
        >
          <p className="font-medium">Please fix the following errors:</p>
          <ul className="mt-1 list-disc list-inside">
            {Object.entries(errors).map(([field, message]) => (
              <li key={field}>{message}</li>
            ))}
          </ul>
        </div>
      )}

      {/* ISBN with Lookup button */}
      <div>
        <label htmlFor="isbn" className={labelClass}>ISBN</label>
        <div className="flex gap-2">
          <input
            id="isbn"
            type="text"
            value={isbn}
            onChange={(e) => {
              setIsbn(e.target.value.replace(/\D/g, ''));
              setLookupError(null);
            }}
            onBlur={(e) => {
              const digits = e.target.value.replace(/\D/g, '');
              if (digits.length === 13) {
                setIsbn(`${digits.slice(0, 3)}-${digits.slice(3, 4)}-${digits.slice(4, 7)}-${digits.slice(7, 12)}-${digits.slice(12)}`);
              } else if (digits.length === 10) {
                setIsbn(`${digits.slice(0, 1)}-${digits.slice(1, 4)}-${digits.slice(4, 9)}-${digits.slice(9)}`);
              }
            }}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault();
                handleLookupISBN();
              }
            }}
            className={inputClass}
            placeholder="978-..."
          />
          <div className="relative group">
            <button
              type="button"
              onClick={handleLookupISBN}
              disabled={lookupLoading || isbn.replace(/\D/g, '').length < 10}
              className="px-4 py-2 bg-primary text-white rounded-md hover:bg-primary/90 transition-colors font-medium disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5 whitespace-nowrap"
              aria-label="Look up book by ISBN. Shift+Click to force refresh."
            >
              {lookupLoading ? (
                <>
                  <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
                  </svg>
                  Looking up...
                </>
              ) : (
                'Lookup'
              )}
            </button>
            <span className="absolute -bottom-5 left-1/2 -translate-x-1/2 text-[10px] text-text-light/50 whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none">
              Shift+Click to force refresh
            </span>
          </div>
        </div>
        {lookupError && (
          <p className="text-error text-xs mt-1" role="alert">{lookupError}</p>
        )}
      </div>

      {/* Title */}
      <div>
        <label htmlFor="title" className={labelClass}>
          Title <span className="text-error" aria-hidden="true">*</span>
        </label>
        <input
          id="title"
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          className={`${inputClass} ${errors.title ? 'border-error ring-1 ring-error' : ''}`}
          placeholder="Enter book title"
          aria-required="true"
          aria-invalid={!!errors.title}
          aria-describedby={errors.title ? 'title-error' : undefined}
        />
        {errors.title && <p id="title-error" className={errorClass} role="alert">{errors.title}</p>}
      </div>

      {/* Subtitle */}
      <div>
        <label htmlFor="subtitle" className={labelClass}>Subtitle</label>
        <input
          id="subtitle"
          type="text"
          value={subtitle}
          onChange={(e) => setSubtitle(e.target.value)}
          className={inputClass}
          placeholder="Optional subtitle"
        />
      </div>

      {/* Authors & Illustrators */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label htmlFor="authors" className={labelClass}>Authors</label>
          <input
            id="authors"
            type="text"
            value={authors}
            onChange={(e) => setAuthors(e.target.value)}
            className={inputClass}
            placeholder="Comma-separated names"
          />
        </div>
        <div>
          <label htmlFor="illustrators" className={labelClass}>Illustrators</label>
          <input
            id="illustrators"
            type="text"
            value={illustrators}
            onChange={(e) => setIllustrators(e.target.value)}
            className={inputClass}
            placeholder="Comma-separated names"
          />
        </div>
      </div>

      {/* Publisher */}
      <div>
        <label htmlFor="publisher" className={labelClass}>Publisher</label>
        <input
          id="publisher"
          type="text"
          value={publisher}
          onChange={(e) => setPublisher(e.target.value)}
          className={inputClass}
          placeholder="Publisher name"
        />
      </div>

      {/* Year & Pages */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label htmlFor="publication_year" className={labelClass}>Publication Year</label>
          <input
            id="publication_year"
            type="number"
            value={publication_year}
            onChange={(e) => setPublicationYear(e.target.value)}
            className={inputClass}
            placeholder="e.g. 2024"
            min={1000}
            max={2100}
          />
        </div>
        <div>
          <label htmlFor="page_count" className={labelClass}>Page Count</label>
          <input
            id="page_count"
            type="number"
            value={page_count}
            onChange={(e) => setPageCount(e.target.value)}
            className={inputClass}
            placeholder="Number of pages"
            min={0}
          />
        </div>
      </div>

      {/* Book Type & Condition */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label htmlFor="book_type" className={labelClass}>Book Type</label>
          <select
            id="book_type"
            value={book_type}
            onChange={(e) => setBookType(e.target.value)}
            className={inputClass}
          >
            {BOOK_TYPES.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label htmlFor="condition" className={labelClass}>Condition</label>
          <select
            id="condition"
            value={condition}
            onChange={(e) => setCondition(e.target.value)}
            className={inputClass}
          >
            {CONDITIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Child Rating */}
      <div>
        <label className={labelClass}>Child Rating</label>
        <div className="flex gap-1">
          {[1, 2, 3, 4, 5].map((star) => (
            <button
              key={star}
              type="button"
              onClick={() => setChildRating(star === child_rating ? 0 : star)}
              className={`text-2xl transition-colors ${
                star <= child_rating ? 'text-accent' : 'text-secondary/30'
              } hover:scale-110`}
              aria-label={`Rate ${star} star${star > 1 ? 's' : ''}`}
            >
              ★
            </button>
          ))}
          {child_rating === 0 && (
            <span className="text-text-light text-sm ml-2 self-center">No rating</span>
          )}
        </div>
      </div>

      {/* Tag input fields */}
      <TagInput
        id="reading_levels"
        label="Reading Levels"
        value={readingLevels}
        onChange={setReadingLevels}
        tagType="reading_levels"
        placeholder="e.g. picture_book, early_reader"
      />

      <TagInput
        id="genres"
        label="Genres"
        value={genres}
        onChange={setGenres}
        tagType="genres"
        placeholder="e.g. fantasy, mystery"
      />

      <TagInput
        id="themes"
        label="Themes"
        value={themes}
        onChange={setThemes}
        tagType="themes"
        placeholder="e.g. friendship, adventure"
      />

      <TagInput
        id="awards"
        label="Awards"
        value={awards}
        onChange={setAwards}
        tagType="awards"
        placeholder="e.g. caldecott_medal, newbery_honor"
      />

      {/* Advanced section toggle */}
      <div className="border-t border-secondary/20 pt-4">
        <button
          type="button"
          onClick={() => setShowAdvanced(!showAdvanced)}
          className="text-primary hover:text-text transition-colors text-sm font-medium flex items-center gap-1"
        >
          <span className={`transform transition-transform ${showAdvanced ? 'rotate-90' : ''}`}>▶</span>
          Advanced Information
        </button>

        {showAdvanced && (
          <div className="mt-4 space-y-4">
            {/* Gift info */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label htmlFor="gift_from" className={labelClass}>Gift From</label>
                <input
                  id="gift_from"
                  type="text"
                  value={gift_from}
                  onChange={(e) => setGiftFrom(e.target.value)}
                  className={inputClass}
                  placeholder="Who gave this book"
                />
              </div>
              <div>
                <label htmlFor="gift_relationship" className={labelClass}>Relationship</label>
                <input
                  id="gift_relationship"
                  type="text"
                  value={gift_relationship}
                  onChange={(e) => setGiftRelationship(e.target.value)}
                  className={inputClass}
                  placeholder="e.g. Grandma, Teacher"
                />
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label htmlFor="date_received" className={labelClass}>Date Received</label>
                <input
                  id="date_received"
                  type="date"
                  value={date_received}
                  onChange={(e) => setDateReceived(e.target.value)}
                  className={inputClass}
                />
              </div>
              <div>
                <label htmlFor="location" className={labelClass}>Location</label>
                <input
                  id="location"
                  type="text"
                  value={location}
                  onChange={(e) => setLocation(e.target.value)}
                  className={inputClass}
                  placeholder="e.g. Shelf A, Bedroom"
                />
              </div>
            </div>

            <div>
              <label htmlFor="cover_image_url" className={labelClass}>Cover Image URL</label>
              <input
                id="cover_image_url"
                type="url"
                value={cover_image_url}
                onChange={(e) => setCoverImageUrl(e.target.value)}
                className={inputClass}
                placeholder="https://..."
              />
            </div>
          </div>
        )}
      </div>

      {/* Notes */}
      <div>
        <label htmlFor="notes" className={labelClass}>Notes</label>
        <textarea
          id="notes"
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
          rows={3}
          className={inputClass}
          placeholder="Any additional notes about this book..."
        />
      </div>

      {/* Actions */}
      <div className="flex justify-end gap-3 pt-4 border-t border-secondary/20">
        <button
          type="button"
          onClick={onCancel}
          className="px-4 py-2 border border-secondary/30 rounded-md text-text hover:bg-background transition-colors"
        >
          Cancel
        </button>
        <button
          type="submit"
          className="px-6 py-2 bg-primary text-white rounded-md hover:bg-primary/90 transition-colors font-medium"
        >
          {isEdit ? 'Save Changes' : 'Add Book'}
        </button>
      </div>
    </form>
  );
}
