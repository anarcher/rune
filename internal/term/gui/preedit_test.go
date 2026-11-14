// Copyright (C) 2017-2026 The Rune Authors
// SPDX-License-Identifier: GPL-3.0-or-later
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or (at
// your option) any later version.
//
// This program is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
// General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package gui

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unstablebuild/rune-go-sdk/term"
	"slices"
	"testing"
)

func TestPreeditOverlay(t *testing.T) {
	for _, tc := range []struct {
		name, text          string
		x, selection, caret int
		widths              []uint8
	}{
		{"hangul", "한글", 1, 6, 5, []uint8{2, 2}},
		{"selection inside composition", "한글", 1, 3, 3, []uint8{2, 2}},
		{"ascii and CJK", "a漢", 0, 4, 3, []uint8{1, 2}},
		{"clip wide character", "한글", 3, 6, 5, []uint8{2}},
		{"no half wide character", "글", 5, 3, 5, nil},
		{"combining cluster", "e\u0301", 1, 3, 2, []uint8{1}},
		{"emoji cluster", "👨‍👧", 1, len("👨‍👧"), 3, []uint8{2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := [][]term.Cell{cellRow("abcdef")}
			before := slices.Clone(src[0])
			var p preedit
			p.set(tc.text, 0, tc.selection)
			p.setAnchor(term.Coordinates{X: tc.x}, true)
			view, caret, active := p.overlay(src)
			require.True(t, active)
			assert.Equal(t, tc.caret, caret.X)
			x := tc.x
			for _, width := range tc.widths {
				assert.Equal(t, width, view[0][x].Width)
				assert.NotZero(t, view[0][x].Attrs&term.AttrUnderline)
				x += int(width)
			}
			assert.Equal(t, before, src[0])
			if len(tc.widths) > 0 && tc.name == "combining cluster" {
				assert.Equal(t, []rune{'\u0301'}, view[0][tc.x].CombiningRunes())
			}
			p.set("", 0, 0)
			restored, _, active := p.overlay(src)
			assert.False(t, active)
			assert.Equal(t, src, restored)
		})
	}
}

func TestPreeditWideCellBoundaries(t *testing.T) {
	for _, x := range []int{0, 1} {
		var p preedit
		p.set("x", 0, 1)
		p.setAnchor(term.Coordinates{X: x}, true)
		source := [][]term.Cell{{{Ch: '한', Width: 2}, {}, {Ch: 'a', Width: 1}}}
		view, _, _ := p.overlay(source)
		assert.Equal(t, 'x', view[0][x].Ch)
		assert.Equal(t, uint8(1), view[0][1-x].Width)
		assert.Equal(t, ' ', view[0][1-x].Ch)
		assert.Equal(t, '한', source[0][0].Ch)
	}
}

func TestPreeditAnchor(t *testing.T) {
	var p preedit
	p.set("글", 0, 3)
	source := [][]term.Cell{cellRow("abcde")}
	_, _, active := p.overlay(source)
	assert.False(t, active)
	p.setAnchor(term.Coordinates{X: 2}, true)
	p.setAnchor(term.Coordinates{}, false)
	_, caret, active := p.overlay(source)
	assert.True(t, active)
	assert.Equal(t, 4, caret.X)
	p.setAnchor(term.Coordinates{X: 99, Y: 99}, true)
	_, _, active = p.overlay(source)
	assert.False(t, active)
}

func TestPreeditDamageRestoresSource(t *testing.T) {
	r, _, _ := newTestRenderer(t, 8, 2)
	t.Cleanup(r.deallocate)
	cells := [][]term.Cell{cellRow("abcdef"), cellRow("abcdef"), cellRow("abcdef"), cellRow("abcdef")}
	r.snapshot(cells, cursorState{})
	var p preedit
	p.setAnchor(term.Coordinates{X: 1}, true)
	p.set("글", 0, 3)
	view, _, _ := p.overlay(cells)
	assert.False(t, r.computeDirtyRows(view, cursorState{}))
	assert.Equal(t, []bool{true, true, false, false}, r.dirtyRows)
	r.snapshot(view, cursorState{})
	p.set("", 0, 0)
	view, _, _ = p.overlay(cells)
	assert.False(t, r.computeDirtyRows(view, cursorState{}))
	assert.Equal(t, []bool{true, true, false, false}, r.dirtyRows)
	assert.Equal(t, cellRow("abcdef"), view[0])
}
