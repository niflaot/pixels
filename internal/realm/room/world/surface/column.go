package surface

import (
	"sort"

	"github.com/niflaot/pixels/internal/realm/room/world/grid"
)

const (
	// inlineSectionLimit stores the sections kept inside a column value.
	inlineSectionLimit = 8
)

// Column stores resolved sections for a tile that has dynamic state.
type Column struct {
	// point stores the tile coordinate.
	point grid.Point

	// version stores the monotonic column version.
	version uint32

	// count stores the number of inline sections.
	count uint8

	// sections stores common tile sections without heap allocation.
	sections [inlineSectionLimit]Section

	// extra stores rare overflow sections.
	extra []Section
}

// NewColumn creates a resolved tile column.
func NewColumn(point grid.Point, version uint32) Column {
	return Column{point: point, version: version}
}

// Point returns the tile coordinate.
func (column Column) Point() grid.Point {
	return column.point
}

// Version returns the column version.
func (column Column) Version() uint32 {
	return column.version
}

// Sections returns the resolved tile sections.
func (column Column) Sections() []Section {
	sections := make([]Section, 0, column.Len())
	for index := 0; index < int(column.count); index++ {
		sections = append(sections, column.sections[index])
	}
	sections = append(sections, column.extra...)
	sortSections(sections)

	return sections
}

// Len returns the number of resolved sections.
func (column Column) Len() int {
	return int(column.count) + len(column.extra)
}

// AddSection adds a resolved tile section.
func (column *Column) AddSection(section Section) {
	if int(column.count) < len(column.sections) {
		column.insertInline(section)

		return
	}

	column.extra = append(column.extra, section)
	sortSections(column.extra)
}

// SectionAt finds a section at the exact walkable height.
func (column Column) SectionAt(height grid.Height) (Section, bool) {
	for index := 0; index < int(column.count); index++ {
		section := column.sections[index]
		if section.Z() == height {
			return section, true
		}
	}
	for _, section := range column.extra {
		if section.Z() == height {
			return section, true
		}
	}

	return Section{}, false
}

// TopSection returns the highest walkable or blocking section.
func (column Column) TopSection() (Section, bool) {
	if column.Len() == 0 {
		return Section{}, false
	}
	top := column.sections[column.count-1]
	if len(column.extra) == 0 {
		return top, true
	}
	extraTop := column.extra[len(column.extra)-1]
	if extraTop.Z() > top.Z() {
		return extraTop, true
	}

	return top, true
}

// Dynamic reports whether the column was materialized from dynamic state.
func (column Column) Dynamic() bool {
	return column.version > 0
}

// insertInline adds an inline section ordered by height.
func (column *Column) insertInline(section Section) {
	position := int(column.count)
	for position > 0 && column.sections[position-1].Z() > section.Z() {
		column.sections[position] = column.sections[position-1]
		position--
	}
	column.sections[position] = section
	column.count++
}

// sortSections orders sections by height.
func sortSections(sections []Section) {
	sort.Slice(sections, func(left int, right int) bool {
		return sections[left].Z() < sections[right].Z()
	})
}
