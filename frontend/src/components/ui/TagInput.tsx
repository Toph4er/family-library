import { useState, useRef, useEffect, useCallback, type KeyboardEvent, type ChangeEvent } from 'react';
import { api } from '@/services/api';

interface TagInputProps {
  id: string;
  label: string;
  value: string[];
  onChange: (tags: string[]) => void;
  tagType: 'genres' | 'themes' | 'awards' | 'reading_levels';
  placeholder?: string;
  error?: string;
}

export default function TagInput({
  id,
  label,
  value,
  onChange,
  tagType,
  placeholder,
  error,
}: TagInputProps) {
  const [inputValue, setInputValue] = useState('');
  const [suggestions, setSuggestions] = useState<string[]>([]);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [highlightedIndex, setHighlightedIndex] = useState(-1);
  const [loading, setLoading] = useState(false);

  const inputRef = useRef<HTMLInputElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Debounced fetch of suggestions from the API
  const fetchSuggestions = useCallback(async (query: string) => {
    if (!query.trim()) {
      setSuggestions([]);
      return;
    }
    setLoading(true);
    try {
      const data = await api.getTags(tagType);
      const allTags: string[] = data.tags || [];
      const lowerQuery = query.toLowerCase();
      const filtered = allTags.filter(
        (t) =>
          t.toLowerCase().includes(lowerQuery) &&
          !value.map((v) => v.toLowerCase()).includes(t.toLowerCase())
      );
      setSuggestions(filtered.slice(0, 10));
    } catch {
      // If the API call fails (e.g., not admin), just show no suggestions
      setSuggestions([]);
    } finally {
      setLoading(false);
    }
  }, [tagType, value]);

  // Debounce the fetch
  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    if (inputValue.trim()) {
      debounceRef.current = setTimeout(() => {
        fetchSuggestions(inputValue);
      }, 250);
    } else {
      setSuggestions([]);
    }
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [inputValue, fetchSuggestions]);

  // Close suggestions when clicking outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setShowSuggestions(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const addTag = (tag: string) => {
    const trimmed = tag.trim();
    if (!trimmed) return;
    // Check for duplicates (case-insensitive)
    if (value.map((v) => v.toLowerCase()).includes(trimmed.toLowerCase())) return;
    onChange([...value, trimmed]);
    setInputValue('');
    setSuggestions([]);
    setShowSuggestions(false);
    setHighlightedIndex(-1);
    inputRef.current?.focus();
  };

  const removeTag = (index: number) => {
    const newValue = [...value];
    newValue.splice(index, 1);
    onChange(newValue);
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    // Enter or comma: add current input as a new tag
    if ((e.key === 'Enter' || e.key === ',') && inputValue.trim()) {
      e.preventDefault();
      addTag(inputValue.replace(/,+$/, ''));
      return;
    }

    // Backspace with empty input: remove last tag
    if (e.key === 'Backspace' && !inputValue && value.length > 0) {
      removeTag(value.length - 1);
      return;
    }

    // Navigate suggestions with arrow keys
    if (e.key === 'ArrowDown' && showSuggestions && suggestions.length > 0) {
      e.preventDefault();
      setHighlightedIndex((prev) => (prev < suggestions.length - 1 ? prev + 1 : 0));
      return;
    }
    if (e.key === 'ArrowUp' && showSuggestions && suggestions.length > 0) {
      e.preventDefault();
      setHighlightedIndex((prev) => (prev > 0 ? prev - 1 : suggestions.length - 1));
      return;
    }
    if (e.key === 'Enter' && showSuggestions && highlightedIndex >= 0) {
      e.preventDefault();
      addTag(suggestions[highlightedIndex]);
      return;
    }
    if (e.key === 'Escape') {
      setShowSuggestions(false);
      setHighlightedIndex(-1);
    }
  };

  const handleInputChange = (e: ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value;
    // If user types a comma, split and add the part before it
    if (val.includes(',')) {
      const parts = val.split(',');
      const lastPart = parts[parts.length - 1];
      const tagToAdd = parts.slice(0, -1).join(',').trim();
      if (tagToAdd) {
        addTag(tagToAdd);
      }
      setInputValue(lastPart);
      return;
    }
    setInputValue(val);
    setShowSuggestions(true);
    setHighlightedIndex(-1);
  };

  const handleSuggestionClick = (suggestion: string) => {
    addTag(suggestion);
  };

  const formatLabel = (tag: string) => tag.replace(/_/g, ' ');

  const labelClass = 'block text-sm font-medium text-text-light mb-1';
  const errorClass = 'text-error text-xs mt-1';

  return (
    <div ref={containerRef} className="relative">
      <label htmlFor={id} className={labelClass}>{label}</label>

      {/* Tag chips + input container */}
      <div
        className={`min-h-[42px] flex flex-wrap items-center gap-1.5 px-3 py-1.5 border rounded-md bg-surface transition-colors ${
          error
            ? 'border-error ring-1 ring-error'
            : 'border-secondary/30 focus-within:ring-2 focus-within:ring-primary/50 focus-within:border-transparent'
        }`}
        onClick={() => inputRef.current?.focus()}
      >
        {value.map((tag, index) => (
          <span
            key={`${tag}-${index}`}
            className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-primary/10 text-primary border border-primary/20"
          >
            {formatLabel(tag)}
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                removeTag(index);
              }}
              className="ml-0.5 text-primary/60 hover:text-primary transition-colors leading-none"
              aria-label={`Remove ${formatLabel(tag)}`}
            >
              &times;
            </button>
          </span>
        ))}

        <input
          ref={inputRef}
          id={id}
          type="text"
          value={inputValue}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          onFocus={() => inputValue.trim() && setShowSuggestions(true)}
          className="flex-1 min-w-[120px] bg-transparent outline-none text-sm text-text placeholder:text-text-light/50"
          placeholder={value.length === 0 ? placeholder || `Type and press Enter to add ${label.toLowerCase()}...` : ''}
          aria-describedby={error ? `${id}-error` : undefined}
          aria-invalid={!!error}
          role="combobox"
          aria-expanded={showSuggestions}
          autoComplete="off"
        />
      </div>

      {error && <p id={`${id}-error`} className={errorClass} role="alert">{error}</p>}

      {/* Suggestions dropdown */}
      {showSuggestions && suggestions.length > 0 && (
        <ul
          className="absolute z-50 mt-1 w-full max-w-md bg-surface border border-secondary/30 rounded-md shadow-lg max-h-60 overflow-y-auto"
          role="listbox"
          aria-label={`${label} suggestions`}
        >
          {suggestions.map((suggestion, index) => (
            <li
              key={suggestion}
              onClick={() => handleSuggestionClick(suggestion)}
              className={`px-3 py-2 cursor-pointer text-sm transition-colors ${
                index === highlightedIndex
                  ? 'bg-primary/10 text-primary'
                  : 'text-text hover:bg-secondary/10'
              }`}
              role="option"
              aria-selected={index === highlightedIndex}
            >
              {formatLabel(suggestion)}
            </li>
          ))}
        </ul>
      )}

      {loading && (
        <div className="absolute right-3 top-8 text-text-light">
          <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
          </svg>
        </div>
      )}
    </div>
  );
}
