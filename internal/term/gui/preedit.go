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
	"unicode/utf8"

	"github.com/unstablebuild/rune-go-sdk/term"
	"github.com/unstablebuild/rune-go-sdk/term/graphemecluster"
)

type preedit struct {
	text        string
	start, end  int
	anchor      term.Coordinates
	anchorValid bool
	rows        [][]term.Cell
	row         []term.Cell
}

func (p *preedit) set(text string, start, end int) bool {
	start = p.byteBoundary(text, start)
	end = max(start, p.byteBoundary(text, end))
	if p.text == text && p.start == start && p.end == end {
		return false
	}
	p.text, p.start, p.end = text, start, end
	return true
}

func (p *preedit) byteBoundary(text string, offset int) int {
	offset = max(0, min(len(text), offset))
	for offset > 0 && offset < len(text) && !utf8.RuneStart(text[offset]) {
		offset--
	}
	return offset
}

func (p *preedit) setAnchor(pos term.Coordinates, show bool) {
	if show {
		p.anchor, p.anchorValid = pos, true
	}
}

func (p *preedit) validAnchor(cells [][]term.Cell) bool {
	return p.anchorValid && p.anchor.Y >= 0 && p.anchor.Y < len(cells) &&
		p.anchor.X >= 0 && p.anchor.X < len(cells[p.anchor.Y])
}

// overlay borrows source rows and copies the composition row. The source grid
// and its pointer-valued extras must survive composition changes untouched.
func (p *preedit) overlay(cells [][]term.Cell) ([][]term.Cell, term.Coordinates, bool) {
	caret := p.anchor
	if p.text == "" || !p.validAnchor(cells) {
		return cells, caret, false
	}
	src := cells[p.anchor.Y]
	p.rows = append(p.rows[:0], cells...)
	p.row = append(p.row[:0], src...)
	p.rows[p.anchor.Y] = p.row
	attr := term.Attributes{Fg: src[p.anchor.X].Fg, Bg: src[p.anchor.X].Bg, Attrs: term.AttrUnderline}
	blank := term.NewCell(' ', 1, term.Attributes{Fg: attr.Fg, Bg: attr.Bg})
	x, offset, state := p.anchor.X, 0, -1
	rest := p.text
	for rest != "" {
		var cluster string
		var width uint8
		cluster, rest, width, state = graphemecluster.StepString(rest, state)
		if width == 0 {
			offset += len(cluster)
			continue
		}
		if x+int(width) > len(src) {
			break
		}
		for col := x; col < x+int(width); col++ {
			p.clearWide(col, blank)
		}
		runes := []rune(cluster)
		c := term.NewCell(runes[0], width, attr)
		c.Bytes = uint8(min(len(cluster), 255))
		c.SetCombining(runes[1:])
		p.row[x] = c
		for col := x + 1; col < x+int(width); col++ {
			p.row[col] = term.Cell{}
		}
		x += int(width)
		offset += len(cluster)
		if offset <= p.end {
			caret.X = x
		}
	}
	caret.X = min(caret.X, len(src)-1)
	return p.rows, caret, true
}

func (p *preedit) clearWide(x int, blank term.Cell) {
	start := x
	if x > 0 && p.row[x-1].Width > 1 {
		start = x - 1
	}
	end := min(len(p.row), start+int(p.row[start].Width))
	if end-start > 1 {
		for col := start; col < end; col++ {
			p.row[col] = blank
		}
	}
}
