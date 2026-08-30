// Package learn 学习中心：读取 content/learning 下的 Markdown 百科/教程与术语词典。
// 内容与代码分离，运营可以直接改 Markdown 增改内容，无需动后端。
package learn

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

var ErrNotFound = errors.New("内容不存在")

type Item struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

type Doc struct {
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type Term struct {
	Term       string `json:"term"`
	En         string `json:"en"`
	Definition string `json:"definition"`
}

type Service struct{ dir string }

func NewService(dir string) *Service { return &Service{dir: dir} }

// parseTitle 从 Markdown 第一个 "# " 标题行提取标题。
func parseTitle(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(t, "# "))
		}
	}
	return ""
}

func (s *Service) list(sub string) ([]Item, error) {
	entries, err := os.ReadDir(filepath.Join(s.dir, sub))
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		slug := strings.TrimSuffix(e.Name(), ".md")
		raw, err := os.ReadFile(filepath.Join(s.dir, sub, e.Name()))
		if err != nil {
			continue
		}
		title := parseTitle(string(raw))
		if title == "" {
			title = slug
		}
		items = append(items, Item{Slug: slug, Title: title})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Slug < items[j].Slug })
	return items, nil
}

func (s *Service) get(sub, slug string) (*Doc, error) {
	if !slugRe.MatchString(slug) {
		return nil, ErrNotFound
	}
	raw, err := os.ReadFile(filepath.Join(s.dir, sub, slug+".md"))
	if err != nil {
		return nil, ErrNotFound
	}
	text := string(raw)
	return &Doc{Slug: slug, Title: parseTitle(text), Content: text}, nil
}

func (s *Service) Coins() ([]Item, error)  { return s.list("coins") }
func (s *Service) Concepts() ([]Item, error) { return s.list("concepts") }

func (s *Service) Coin(slug string) (*Doc, error)    { return s.get("coins", slug) }
func (s *Service) Concept(slug string) (*Doc, error) { return s.get("concepts", slug) }

func (s *Service) Glossary() ([]Term, error) {
	raw, err := os.ReadFile(filepath.Join(s.dir, "glossary.json"))
	if err != nil {
		return nil, err
	}
	var terms []Term
	if err := json.Unmarshal(raw, &terms); err != nil {
		return nil, err
	}
	return terms, nil
}
