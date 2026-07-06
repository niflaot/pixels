package live

import (
	"sync"
	"time"
)

// LastSearch stores the viewer's last navigator query.
type LastSearch struct {
	// Code stores the search context or result code.
	Code string

	// Query stores the search query.
	Query string
}

// Viewer stores active navigator UI state for one player.
type Viewer struct {
	// mutex protects viewer state.
	mutex sync.RWMutex

	// initializedAt stores when the viewer initialized.
	initializedAt time.Time

	// lastSearch stores the last navigator search.
	lastSearch LastSearch

	// categoryCounts reports whether category count updates are enabled.
	categoryCounts bool
}

// NewViewer creates a navigator viewer.
func NewViewer() *Viewer {
	return &Viewer{initializedAt: time.Now(), categoryCounts: true}
}

// SetLastSearch replaces the viewer last search.
func (viewer *Viewer) SetLastSearch(search LastSearch) {
	viewer.mutex.Lock()
	defer viewer.mutex.Unlock()

	viewer.lastSearch = search
}

// LastSearch returns the viewer last search.
func (viewer *Viewer) LastSearch() LastSearch {
	viewer.mutex.RLock()
	defer viewer.mutex.RUnlock()

	return viewer.lastSearch
}

// SetCategoryCounts enables or disables category count updates.
func (viewer *Viewer) SetCategoryCounts(enabled bool) {
	viewer.mutex.Lock()
	defer viewer.mutex.Unlock()

	viewer.categoryCounts = enabled
}

// ReceivesCategoryCounts reports whether the viewer receives category counts.
func (viewer *Viewer) ReceivesCategoryCounts() bool {
	viewer.mutex.RLock()
	defer viewer.mutex.RUnlock()

	return viewer.categoryCounts
}
