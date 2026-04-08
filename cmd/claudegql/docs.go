package main

import (
	"bytes"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"sort"
	"strings"

	docspkg "github.com/wricardo/claude-code-graphql/docs"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	htmlrenderer "github.com/yuin/goldmark/renderer/html"
)

var (
	docsMD       goldmark.Markdown
	docsSections []docsSection
	docsIndex    map[string]*docsEntry
	docsFirst    string
)

type docsEntry struct {
	title string
	slug  string
	rawMD []byte
}

type docsSection struct {
	Title string
	Pages []docsNavPage
}

type docsNavPage struct {
	Title string
	Href  string
}

type docsTemplateData struct {
	Title       string
	Content     template.HTML
	Sections    []docsSection
	CurrentHref string
	PrevPage    *docsNavPage
	NextPage    *docsNavPage
}

var numPrefix = regexp.MustCompile(`^\d+-`)

func slugFromFilename(name string) string {
	name = strings.TrimSuffix(name, ".md")
	name = numPrefix.ReplaceAllString(name, "")
	return name
}

func titleFromMD(src []byte, fallback string) string {
	for _, line := range strings.Split(string(src), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	s := strings.ReplaceAll(fallback, "-", " ")
	if len(s) > 0 {
		return strings.ToUpper(s[:1]) + s[1:]
	}
	return fallback
}

func sectionTitle(dir string) string {
	dir = numPrefix.ReplaceAllString(dir, "")
	dir = strings.ReplaceAll(dir, "-", " ")
	if len(dir) > 0 {
		return strings.ToUpper(dir[:1]) + dir[1:]
	}
	return dir
}

func init() {
	docsMD = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Typographer,
			highlighting.NewHighlighting(
				highlighting.WithStyle("dracula"),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			htmlrenderer.WithUnsafe(),
		),
	)

	docsSections, docsIndex, docsFirst = buildDocsNav()
}

func buildDocsNav() ([]docsSection, map[string]*docsEntry, string) {
	idx := make(map[string]*docsEntry)
	var sections []docsSection
	var firstSlug string

	topEntries, _ := fs.ReadDir(docspkg.FS, ".")
	var topFiles []fs.DirEntry
	var dirs []fs.DirEntry
	for _, e := range topEntries {
		if e.IsDir() {
			dirs = append(dirs, e)
		} else if strings.HasSuffix(e.Name(), ".md") {
			topFiles = append(topFiles, e)
		}
	}
	sort.Slice(topFiles, func(i, j int) bool { return topFiles[i].Name() < topFiles[j].Name() })
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })

	var topPages []docsNavPage
	for _, f := range topFiles {
		slug := slugFromFilename(f.Name())
		raw, _ := docspkg.FS.ReadFile(f.Name())
		title := titleFromMD(raw, slug)
		href := "/docs/" + slug
		entry := &docsEntry{title: title, slug: slug, rawMD: raw}
		idx[slug] = entry
		topPages = append(topPages, docsNavPage{Title: title, Href: href})
		if firstSlug == "" {
			firstSlug = slug
		}
	}
	if len(topPages) > 0 {
		sections = append(sections, docsSection{Title: "", Pages: topPages})
	}

	for _, dir := range dirs {
		dirName := dir.Name()
		subEntries, _ := fs.ReadDir(docspkg.FS, dirName)
		var pages []docsNavPage
		for _, f := range subEntries {
			if !strings.HasSuffix(f.Name(), ".md") {
				continue
			}
			fileSlug := slugFromFilename(f.Name())
			urlDir := numPrefix.ReplaceAllString(dirName, "")
			urlSlug := urlDir + "/" + fileSlug
			raw, _ := docspkg.FS.ReadFile(path.Join(dirName, f.Name()))
			title := titleFromMD(raw, fileSlug)
			href := "/docs/" + urlSlug
			entry := &docsEntry{title: title, slug: urlSlug, rawMD: raw}
			idx[urlSlug] = entry
			pages = append(pages, docsNavPage{Title: title, Href: href})
			if firstSlug == "" {
				firstSlug = urlSlug
			}
		}
		if len(pages) > 0 {
			sections = append(sections, docsSection{
				Title: sectionTitle(dirName),
				Pages: pages,
			})
		}
	}

	return sections, idx, firstSlug
}

func flatNavPages() []docsNavPage {
	var all []docsNavPage
	for _, sec := range docsSections {
		all = append(all, sec.Pages...)
	}
	return all
}

func newDocsHandler() http.Handler {
	tmpl := template.Must(template.ParseFS(uiFS, "ui/docs.html"))
	all := flatNavPages()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		urlPath := strings.TrimPrefix(r.URL.Path, "/docs")
		urlPath = strings.TrimPrefix(urlPath, "/")

		if urlPath == "" {
			if docsFirst != "" {
				http.Redirect(w, r, "/docs/"+docsFirst, http.StatusFound)
			} else {
				http.NotFound(w, r)
			}
			return
		}

		entry, ok := docsIndex[urlPath]
		if !ok {
			http.NotFound(w, r)
			return
		}

		var buf bytes.Buffer
		if err := docsMD.Convert(entry.rawMD, &buf); err != nil {
			http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		href := "/docs/" + urlPath
		var prev, next *docsNavPage
		for i, p := range all {
			if p.Href == href {
				if i > 0 {
					prev = &all[i-1]
				}
				if i < len(all)-1 {
					next = &all[i+1]
				}
				break
			}
		}

		data := docsTemplateData{
			Title:       entry.title,
			Content:     template.HTML(buf.String()),
			Sections:    docsSections,
			CurrentHref: href,
			PrevPage:    prev,
			NextPage:    next,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		}
	})
}
