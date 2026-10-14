package models

import "testing"

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

// setAllPointerFields sets every pointer field on the book to a distinct
// non-nil value so tests can assert exactly which ones ApplyVisibility nils.
func setAllPointerFields(b *Book) {
	b.ISBN = strPtr("978-0-00-000000-1")
	b.Subtitle = strPtr("A Subtitle")
	b.Authors = strPtr(`["Ada Lovelace"]`)
	b.Illustrators = strPtr(`["Ivy"]`)
	b.Publisher = strPtr("Acme Press")
	b.PublicationYear = intPtr(2020)
	b.PageCount = intPtr(120)
	b.BookType = strPtr("picture book")
	b.ReadingLevels = strPtr(`["3-5"]`)
	b.Genres = strPtr(`["adventure"]`)
	b.Themes = strPtr(`["friendship"]`)
	b.Awards = strPtr(`["Caldecott"]`)
	b.GiftFrom = strPtr("Grandma")
	b.GiftRelationship = strPtr("grandmother")
	b.DateReceived = strPtr("2024-01-15")
	b.Condition = strPtr("good")
	b.Location = strPtr("Shelf A")
	b.Notes = strPtr("SECRET-NOTES")
	b.ChildRating = intPtr(5)
	b.LastReadDate = strPtr("2024-06-01")
	b.CoverImageURL = strPtr("https://x/c.png")
	b.CoverSource = strPtr("gutenberg")
	b.DeweyDecimalClass = strPtr("700")
	b.Description = strPtr("desc")
	b.Language = strPtr("en")
	b.SubjectPlaces = strPtr(`["Rome"]`)
	b.SubjectPeople = strPtr(`["Cat"]`)
	b.SubjectTimes = strPtr(`["1900"]`)
	b.Series = strPtr("Series X")
	b.AgeRange = strPtr("4-8")
}

// allVisible is a visibility map that marks every pointer field visible.
func allVisible() map[string]bool {
	return map[string]bool{
		"isbn": true, "subtitle": true, "authors": true, "illustrators": true,
		"publisher": true, "publication_year": true, "page_count": true,
		"book_type": true, "reading_levels": true, "genres": true,
		"themes": true, "awards": true, "gift_from": true,
		"gift_relationship": true, "date_received": true,
		"condition": true, "child_rating": true, "last_read_date": true,
		"cover_image_url": true, "cover_source": true,
		"dewey_decimal_class": true, "description": true, "language": true,
		"subject_places": true, "subject_people": true, "subject_times": true,
		"series": true, "age_range": true,
	}
}

func TestBookApplyVisibility_HidesHiddenPointerFields(t *testing.T) {
	b := &Book{Title: "Test"}
	setAllPointerFields(b)

	// Start from all-visible, then hide three specific fields.
	vis := allVisible()
	vis["notes"] = false
	vis["isbn"] = false
	vis["location"] = false
	b.ApplyVisibility(vis)

	if b.Notes != nil {
		t.Errorf("Notes should be nil (hidden), got %q", *b.Notes)
	}
	if b.ISBN != nil {
		t.Errorf("ISBN should be nil (hidden), got %q", *b.ISBN)
	}
	if b.Location != nil {
		t.Errorf("Location should be nil (hidden), got %q", *b.Location)
	}
	// Visible fields must be preserved.
	if b.Authors == nil {
		t.Error("Authors should be preserved (visible)")
	}
	if b.ChildRating == nil {
		t.Error("ChildRating should be preserved (visible)")
	}
	if b.Title != "Test" {
		t.Error("Title (non-pointer) must be untouched")
	}
}

func TestBookApplyVisibility_AbsentKeyIsHidden(t *testing.T) {
	// A key absent from the map must be treated as hidden — this matches the
	// settings UI where unchecked fields are hidden.
	b := &Book{}
	setAllPointerFields(b)
	b.ApplyVisibility(map[string]bool{"title": true}) // only title listed

	if b.Notes != nil || b.ISBN != nil || b.Authors != nil || b.Location != nil {
		t.Errorf("absent keys must be hidden; Notes=%v ISBN=%v Authors=%v Location=%v",
			b.Notes, b.ISBN, b.Authors, b.Location)
	}
}

func TestBookApplyVisibility_NilMapIsNoOp(t *testing.T) {
	b := &Book{}
	setAllPointerFields(b)
	b.ApplyVisibility(nil)
	if b.Notes == nil || b.Authors == nil || b.ISBN == nil {
		t.Error("nil visibility map must not hide any field")
	}
}

func TestBookApplyVisibility_EmptyMapHidesAllPointer(t *testing.T) {
	// An empty map means nothing is explicitly visible → all pointer fields
	// hidden (the admin unchecked every box).
	b := &Book{}
	setAllPointerFields(b)
	b.ApplyVisibility(map[string]bool{})
	if b.Notes != nil || b.Authors != nil || b.ISBN != nil || b.Genres != nil {
		t.Errorf("empty map must hide all pointer fields; Notes=%v Authors=%v ISBN=%v Genres=%v",
			b.Notes, b.Authors, b.ISBN, b.Genres)
	}
	if b.Title != "" || b.Quantity != 0 || b.ReadCount != 0 {
		t.Error("non-pointer fields must remain at their zero values")
	}
}

func TestBookApplyVisibility_NonPointerIdentityUntouched(t *testing.T) {
	b := &Book{Title: "Keep Me", Quantity: 3, ReadCount: 7}
	setAllPointerFields(b)
	b.ApplyVisibility(map[string]bool{}) // hide every pointer field
	if b.Title != "Keep Me" || b.Quantity != 3 || b.ReadCount != 7 {
		t.Errorf("non-pointer identity fields must be untouched: %q/%d/%d", b.Title, b.Quantity, b.ReadCount)
	}
}

func TestFilterForGuest_UsesPerBookBlob(t *testing.T) {
	b := &Book{}
	setAllPointerFields(b)
	// Per-book blob: hide notes, show authors; isbn absent → hidden.
	b.GuestVisibleFields = `{"notes":false,"authors":true}`
	b.FilterForGuest()
	if b.Notes != nil {
		t.Error("per-book blob: Notes should be hidden")
	}
	if b.Authors == nil {
		t.Error("per-book blob: Authors should be shown")
	}
	if b.ISBN != nil {
		t.Error("per-book blob: absent ISBN should be hidden")
	}
}

func TestFilterForGuest_EmptyBlobNoChange(t *testing.T) {
	b := &Book{}
	setAllPointerFields(b)
	b.GuestVisibleFields = ""
	b.FilterForGuest()
	if b.Notes == nil || b.Authors == nil {
		t.Error("empty blob must leave the book untouched")
	}
}

func TestFilterForGuest_MalformedBlobNoChange(t *testing.T) {
	b := &Book{}
	setAllPointerFields(b)
	b.GuestVisibleFields = `not-json`
	b.FilterForGuest()
	if b.Notes == nil || b.Authors == nil {
		t.Error("malformed blob must leave the book untouched")
	}
}
