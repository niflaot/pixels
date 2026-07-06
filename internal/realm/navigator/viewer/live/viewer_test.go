package live

import "testing"

// TestViewerStoresLastSearch verifies viewer search state.
func TestViewerStoresLastSearch(t *testing.T) {
	viewer := NewViewer()
	viewer.SetLastSearch(LastSearch{Code: "hotel_view", Query: "demo"})

	search := viewer.LastSearch()
	if search.Code != "hotel_view" || search.Query != "demo" {
		t.Fatalf("unexpected search %#v", search)
	}
}

// TestViewerCategoryCounts verifies category count preference state.
func TestViewerCategoryCounts(t *testing.T) {
	viewer := NewViewer()
	if !viewer.ReceivesCategoryCounts() {
		t.Fatal("expected category counts by default")
	}

	viewer.SetCategoryCounts(false)
	if viewer.ReceivesCategoryCounts() {
		t.Fatal("expected category counts disabled")
	}
}
