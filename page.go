// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package webki provides a framework designed for easily building content-focused sites
package webki

import (
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"strings"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/girl/styles"
	"goki.dev/glide/gidom"
	"goki.dev/glop/dirs"
	"goki.dev/goosi"
	"goki.dev/goosi/events"
	"goki.dev/grr"
	"goki.dev/ki/v2"
)

// Page represents one site page
type Page struct {
	gi.Frame

	// Source is the filesystem in which the content is located.
	Source fs.FS

	// The history of URLs that have been visited. The oldest page is first.
	History []string

	// HistoryIndex is the current place we are at in the History
	HistoryIndex int

	// PageURL is the current page URL
	PageURL string

	// PagePath is the fs path of the current page in [Page.Source]
	PagePath string
}

var _ ki.Ki = (*Page)(nil)

// OpenURL sets the content of the page from the given url. If the given URL
// has no scheme (eg: "/about"), then it sets the content of the page to the
// file specified by the URL. This is either the "index.md" file in the
// corresponding directory (eg: "/about/index.md") or the corresponding
// md file (eg: "/about.md"). If it has a scheme, (eg: "https://example.com"),
// then it opens it in the user's default browser.
func (pg *Page) OpenURL(rawURL string, addToHistory bool) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if u.Scheme != "" {
		goosi.TheApp.OpenURL(u.String())
		return nil
	}

	if pg.Source == nil {
		return fmt.Errorf("page source must not be nil")
	}

	// if we are not rooted, we go relative to our current fs path
	if !strings.HasPrefix(rawURL, "/") {
		rawURL = path.Join(path.Dir(pg.PagePath), rawURL)
	}

	// the paths in the fs are never rooted, so we trim a rooted one
	rawURL = strings.TrimPrefix(rawURL, "/")

	pg.PageURL = rawURL
	if addToHistory {
		pg.HistoryIndex = len(pg.History)
		pg.History = append(pg.History, pg.PageURL)
	}

	fsPath := path.Join(rawURL, "index.md")
	exists, err := dirs.FileExistsFS(pg.Source, fsPath)
	if err != nil {
		return err
	}
	if !exists {
		fsPath = path.Clean(rawURL) + ".md"
	}

	pg.PagePath = fsPath

	b, err := fs.ReadFile(pg.Source, fsPath)
	if err != nil {
		return err
	}

	fr := pg.FindPath("splits/body").(*gi.Frame)
	updt := fr.UpdateStart()
	fr.DeleteChildren(true)
	err = gidom.ReadMD(pg.Context(), fr, b)
	if err != nil {
		return err
	}
	fr.Update()
	fr.UpdateEndLayout(updt)
	return nil
}

func (pg *Page) ConfigWidget() {
	if pg.HasChildren() {
		return
	}

	updt := pg.UpdateStart()
	sp := gi.NewSplits(pg, "splits")

	nav := giv.NewTreeView(sp, "nav")
	nav.SetText("Nav")

	gi.NewFrame(sp, "body").Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	sp.SetSplits(0.2, 0.8)
	pg.UpdateEnd(updt)
}

// TopAppBar is the default [gi.TopAppBar] for a [Page]
func (pg *Page) TopAppBar(tb *gi.TopAppBar) {
	gi.DefaultTopAppBarStd(tb)

	ch := tb.ChildByName("nav-bar").(*gi.Chooser)

	back := tb.ChildByName("back").(*gi.Button)
	back.OnClick(func(e events.Event) {
		if pg.HistoryIndex > 0 {
			pg.HistoryIndex--
			// we reverse the order
			// ch.SelectItem(len(pg.History) - pg.HistoryIndex - 1)
			// we need a slash so that it doesn't think it's a relative URL
			pg.OpenURL("/"+pg.History[pg.HistoryIndex], false)
		}
	})

	ch.AllowNew = true
	ch.ItemsFunc = func() {
		ch.Items = make([]any, len(pg.History))
		for i, u := range pg.History {
			// we reverse the order
			ch.Items[len(pg.History)-i-1] = u
		}
	}
	ch.OnChange(func(e events.Event) {
		// we need a slash so that it doesn't think it's a relative URL
		grr.Log0(pg.OpenURL("/"+ch.CurLabel, true))
		e.SetHandled()
	})
}
