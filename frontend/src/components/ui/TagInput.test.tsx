import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/react';
import TagInput from './TagInput';

// Mock the API
vi.mock('@/services/api', () => ({
  api: {
    getTags: vi.fn(),
  },
}));

const getMockedApi = async () => {
  const mod = await import('@/services/api');
  return mod.api;
};

function renderTagInput({
  value = [],
  onChange = vi.fn(),
  tagType = 'genres',
  id = 'test-tags',
  label = 'Genres',
  placeholder,
}: {
  value?: string[];
  onChange?: ReturnType<typeof vi.fn>;
  tagType?: 'genres' | 'themes' | 'awards' | 'reading_levels';
  id?: string;
  label?: string;
  placeholder?: string;
}) {
  return render(
    <TagInput
      id={id}
      label={label}
      value={value}
      onChange={onChange}
      tagType={tagType}
      placeholder={placeholder}
    />
  );
}

describe('TagInput', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  describe('tag display', () => {
    it('renders the label', () => {
      renderTagInput({ label: 'Genres' });
      expect(screen.getByText('Genres')).toBeInTheDocument();
    });

    it('displays existing tags as chips', () => {
      renderTagInput({ value: ['Fantasy', 'Science Fiction'] });
      expect(screen.getByText('Fantasy')).toBeInTheDocument();
      expect(screen.getByText('Science Fiction')).toBeInTheDocument();
    });

    it('renders a remove button for each tag', () => {
      renderTagInput({ value: ['Fantasy', 'Mystery'] });
      const removeButtons = screen.getAllByRole('button');
      expect(removeButtons).toHaveLength(2);
    });

    it('formats tag labels by replacing underscores with spaces', () => {
      renderTagInput({ value: ['science_fiction', 'fantasy_adventure'] });
      expect(screen.getByText('science fiction')).toBeInTheDocument();
      expect(screen.getByText('fantasy adventure')).toBeInTheDocument();
    });
  });

  describe('tag removal', () => {
    it('removes a tag when its remove button is clicked', async () => {
      const onChange = vi.fn();
      renderTagInput({ value: ['Fantasy', 'Mystery'], onChange });

      const removeButtons = screen.getAllByRole('button');
      fireEvent.click(removeButtons[0]);

      expect(onChange).toHaveBeenCalledWith(['Mystery']);
    });

    it('removes the correct tag by index', async () => {
      const onChange = vi.fn();
      renderTagInput({ value: ['A', 'B', 'C'], onChange });

      const removeButtons = screen.getAllByRole('button');
      // Remove the middle tag (B)
      fireEvent.click(removeButtons[1]);

      expect(onChange).toHaveBeenCalledWith(['A', 'C']);
    });
  });

  describe('tag addition via keyboard', () => {
    it('adds a tag when Enter is pressed', async () => {
      const onChange = vi.fn();
      renderTagInput({ value: [], onChange });

      const input = screen.getByRole('combobox') as HTMLInputElement;
      fireEvent.change(input, { target: { value: 'Fantasy' } });
      fireEvent.keyDown(input, { key: 'Enter' });

      expect(onChange).toHaveBeenCalledWith(['Fantasy']);
    });

    it('adds a tag when comma is pressed', async () => {
      const onChange = vi.fn();
      renderTagInput({ value: [], onChange });

      const input = screen.getByRole('combobox') as HTMLInputElement;
      fireEvent.change(input, { target: { value: 'Fantasy' } });
      fireEvent.keyDown(input, { key: ',' });

      expect(onChange).toHaveBeenCalledWith(['Fantasy']);
    });

    it('does not add empty tags', async () => {
      const onChange = vi.fn();
      renderTagInput({ value: [], onChange });

      const input = screen.getByRole('combobox') as HTMLInputElement;
      fireEvent.change(input, { target: { value: '   ' } });
      fireEvent.keyDown(input, { key: 'Enter' });

      expect(onChange).not.toHaveBeenCalled();
    });

    it('does not add duplicate tags (case-insensitive)', async () => {
      const onChange = vi.fn();
      renderTagInput({ value: ['Fantasy'], onChange });

      const input = screen.getByRole('combobox') as HTMLInputElement;
      fireEvent.change(input, { target: { value: 'fantasy' } });
      fireEvent.keyDown(input, { key: 'Enter' });

      expect(onChange).not.toHaveBeenCalled();
    });

    it('trims whitespace from added tags', async () => {
      const onChange = vi.fn();
      renderTagInput({ value: [], onChange });

      const input = screen.getByRole('combobox') as HTMLInputElement;
      fireEvent.change(input, { target: { value: '  Fantasy  ' } });
      fireEvent.keyDown(input, { key: 'Enter' });

      expect(onChange).toHaveBeenCalledWith(['Fantasy']);
    });
  });

  describe('tag addition via comma in input', () => {
    it('splits on comma and adds the tag before it', async () => {
      const onChange = vi.fn();
      renderTagInput({ value: [], onChange });

      const input = screen.getByRole('combobox') as HTMLInputElement;
      fireEvent.change(input, { target: { value: 'Fantasy,' } });

      expect(onChange).toHaveBeenCalledWith(['Fantasy']);
    });

    it('keeps text after the comma in the input', async () => {
      const onChange = vi.fn();
      renderTagInput({ value: [], onChange });

      const input = screen.getByRole('combobox') as HTMLInputElement;
      fireEvent.change(input, { target: { value: 'Fantasy,Mystery' } });

      expect(onChange).toHaveBeenCalledWith(['Fantasy']);
      expect(input).toHaveValue('Mystery');
    });
  });

  describe('backspace behavior', () => {
    it('removes the last tag when backspace is pressed with empty input', async () => {
      const onChange = vi.fn();
      renderTagInput({ value: ['Fantasy', 'Mystery'], onChange });

      const input = screen.getByRole('combobox') as HTMLInputElement;
      // Input is already empty
      fireEvent.keyDown(input, { key: 'Backspace' });

      expect(onChange).toHaveBeenCalledWith(['Fantasy']);
    });

    it('does not remove a tag when input has text', async () => {
      const onChange = vi.fn();
      renderTagInput({ value: ['Fantasy'], onChange });

      const input = screen.getByRole('combobox') as HTMLInputElement;
      fireEvent.change(input, { target: { value: 'some text' } });
      fireEvent.keyDown(input, { key: 'Backspace' });

      expect(onChange).not.toHaveBeenCalled();
    });
  });

  describe('suggestion fetching', () => {
    it('fetches suggestions when user types', async () => {
      const api = await getMockedApi();
      (api.getTags as ReturnType<typeof vi.fn>).mockResolvedValue({
        tags: ['Fantasy', 'Fantasy Adventure', 'Dark Fantasy'],
      });

      renderTagInput({ value: [], tagType: 'genres' });

      const input = screen.getByRole('combobox') as HTMLInputElement;
      fireEvent.change(input, { target: { value: 'Fantasy' } });

      await waitFor(() => {
        expect(api.getTags).toHaveBeenCalledWith('genres');
      });
    });

    it('does not fetch suggestions for empty input', async () => {
      const api = await getMockedApi();
      (api.getTags as ReturnType<typeof vi.fn>).mockResolvedValue({ tags: [] });

      renderTagInput({ value: [] });

      const input = screen.getByRole('combobox') as HTMLInputElement;
      fireEvent.change(input, { target: { value: '' } });

      // Wait a bit to ensure no fetch was made
      await new Promise((r) => setTimeout(r, 300));
      expect(api.getTags).not.toHaveBeenCalled();
    });

    it('filters suggestions to exclude already-selected tags', async () => {
      const api = await getMockedApi();
      (api.getTags as ReturnType<typeof vi.fn>).mockResolvedValue({
        tags: ['Fantasy', 'Mystery', 'Science Fiction'],
      });

      renderTagInput({ value: ['Fantasy'] });

      const input = screen.getByRole('combobox') as HTMLInputElement;
      fireEvent.change(input, { target: { value: 'f' } });

      await waitFor(() => {
        // The listbox should exist with suggestions
        const listbox = screen.getByRole('listbox');
        expect(listbox).toBeInTheDocument();
        // Fantasy should NOT appear as a suggestion option
        const options = listbox.querySelectorAll('[role="option"]');
        const optionTexts = Array.from(options).map((o) => o.textContent);
        expect(optionTexts).not.toContain('Fantasy');
      });
    });

    it('handles API errors gracefully without crashing', async () => {
      const api = await getMockedApi();
      (api.getTags as ReturnType<typeof vi.fn>).mockRejectedValue(
        new Error('Not authorized')
      );

      renderTagInput({ value: [] });

      const input = screen.getByRole('combobox') as HTMLInputElement;
      fireEvent.change(input, { target: { value: 'test' } });

      // Should not throw, just show no suggestions
      await waitFor(() => {
        expect(screen.queryByRole('listbox')).not.toBeInTheDocument();
      });
    });
  });

  describe('suggestion dropdown', () => {
    it('shows suggestions dropdown when suggestions are available', async () => {
      const api = await getMockedApi();
      (api.getTags as ReturnType<typeof vi.fn>).mockResolvedValue({
        tags: ['Fantasy', 'Mystery'],
      });

      renderTagInput({ value: [] });

      const input = screen.getByRole('combobox') as HTMLInputElement;
      fireEvent.change(input, { target: { value: 'F' } });

      await waitFor(() => {
        expect(screen.getByRole('listbox')).toBeInTheDocument();
      });
    });

    it('adds a suggestion when clicked', async () => {
      const onChange = vi.fn();
      const api = await getMockedApi();
      (api.getTags as ReturnType<typeof vi.fn>).mockResolvedValue({
        tags: ['Fantasy', 'Mystery'],
      });

      renderTagInput({ value: [], onChange });

      const input = screen.getByRole('combobox') as HTMLInputElement;
      fireEvent.change(input, { target: { value: 'F' } });

      await waitFor(() => {
        expect(screen.getByRole('listbox')).toBeInTheDocument();
      });

      const suggestion = screen.getByRole('option', { name: 'Fantasy' });
      fireEvent.click(suggestion);

      expect(onChange).toHaveBeenCalledWith(['Fantasy']);
    });

    it('hides suggestions when Escape is pressed', async () => {
      const api = await getMockedApi();
      (api.getTags as ReturnType<typeof vi.fn>).mockResolvedValue({
        tags: ['Fantasy', 'Mystery'],
      });

      renderTagInput({ value: [] });

      const input = screen.getByRole('combobox') as HTMLInputElement;
      fireEvent.change(input, { target: { value: 'F' } });

      await waitFor(() => {
        expect(screen.getByRole('listbox')).toBeInTheDocument();
      });

      fireEvent.keyDown(input, { key: 'Escape' });

      expect(screen.queryByRole('listbox')).not.toBeInTheDocument();
    });
  });

  describe('error display', () => {
    it('renders error message when provided', () => {
      render(
        <TagInput
          id="test-tags"
          label="Genres"
          value={[]}
          onChange={() => {}}
          tagType="genres"
          error="Please select at least one genre"
        />
      );

      expect(screen.getByRole('alert')).toHaveTextContent(
        'Please select at least one genre'
      );
    });

    it('sets aria-invalid when error is present', () => {
      render(
        <TagInput
          id="test-tags"
          label="Genres"
          value={[]}
          onChange={() => {}}
          tagType="genres"
          error="Required field"
        />
      );

      const input = screen.getByRole('combobox');
      expect(input).toHaveAttribute('aria-invalid', 'true');
    });

    it('links error message to input via aria-describedby', () => {
      render(
        <TagInput
          id="test-tags"
          label="Genres"
          value={[]}
          onChange={() => {}}
          tagType="genres"
          error="Required field"
        />
      );

      const input = screen.getByRole('combobox');
      expect(input).toHaveAttribute('aria-describedby', 'test-tags-error');
    });
  });

  describe('placeholder', () => {
    it('shows custom placeholder when provided', () => {
      renderTagInput({ placeholder: 'Enter a genre...' });
      const input = screen.getByRole('combobox');
      expect(input).toHaveAttribute('placeholder', 'Enter a genre...');
    });

    it('shows default placeholder when no custom placeholder', () => {
      renderTagInput({});
      const input = screen.getByRole('combobox');
      expect(input).toHaveAttribute('placeholder', 'Type and press Enter to add genres...');
    });

    it('hides placeholder when tags are present', () => {
      renderTagInput({ value: ['Fantasy'] });
      const input = screen.getByRole('combobox');
      expect(input).toHaveAttribute('placeholder', '');
    });
  });
});
