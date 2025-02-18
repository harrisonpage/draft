package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"

	"github.com/gomarkdown/markdown/parser"
	"gopkg.in/yaml.v3"
)

var Version string
var BuildDate string

type PostStatus string

const (
	Public  PostStatus = "public"
	Private PostStatus = "private"
)

var ValidPostStatuses = map[PostStatus]struct{}{
	Public:  {},
	Private: {},
}

type FrontMatter struct {
	Title       string   `yaml:"title"`
	Link        string   `yaml:"link"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
	Image       string   `yaml:"image"`
	Alt         string   `yaml:"alt"`
	Published   string   `yaml:"published"`
	Template    string   `yaml:"template"`
	Favicon     string   `yaml:"favicon"`
	Author      string   `yaml:"author"`
	Email       string   `yaml:"email"`
	Status      string   `yaml:"status"`
	Related     []string `yaml:"related"`
}

/*
 * Support for external search engines
 */
type SearchConfig struct {
	Enabled bool   `yaml:"enabled"`
	Engine  string `yaml:"engine"`
	URL     string `yaml:"url"`
	Path    string `yaml:"path"`
	Dir     string `yaml:"dir"`
}

/*
 * Fields in config.yaml
 */
type Config struct {
	InputDir              string       `yaml:"input_dir"`
	TemplatesDir          string       `yaml:"templates_dir"`
	OutputDir             string       `yaml:"output_dir"`
	BadgesDir             string       `yaml:"badges_dir"`
	IndexTemplatePath     string       `yaml:"index_template_path"`
	TagsIndexTemplatePath string       `yaml:"tags_index_template_path"`
	TagPageTemplatePath   string       `yaml:"tag_page_template_path"`
	Author                string       `yaml:"author"`
	BlogName              string       `yaml:"blog_name"`
	Description           string       `yaml:"description"`
	Email                 string       `yaml:"email"`
	Language              string       `yaml:"language"`
	Locale                string       `yaml:"locale"`
	Lang                  string       `yaml:"lang"`
	BackLabel             string       `yaml:"back_label"`
	CSSFiles              []string     `yaml:"css_files"`
	JSFiles               []string     `yaml:"js_files"`
	Pages                 []Page       `yaml:"pages"`
	URL                   string       `yaml:"url"`
	BasePath              string       `yaml:"base_path"`
	Badges                []Badge      `yaml:"badges"`
	FediverseCreator      string       `yaml:"fediverse_creator"`
	Search                SearchConfig `yaml:"search"`
	Rights                string       `yaml:"rights"`
}

type Badge struct {
	Title string
	URL   string
	Icon  string
	ID    string
}

type Page struct {
	Template string
	Title    string
	Link     string
}

type Labels struct {
	Title string
}

type Unfurl struct {
	Title       string
	URL         string
	Author      string
	Description string
	SiteName    string
	Tags        string
	Locale      string
}

type Links struct {
	Home    string
	Tags    string
	RSS     string
	Atom    string
	Sitemap string
	Rights  string
}

type Tag struct {
	TagName string
	URL     string
}

type Post struct {
	FrontMatter FrontMatter
	URL         string
	HTML        string
	Text        string    // Plaintext representation
	PubTime     time.Time // parsed version of Published date
	PubDate     string    // 15-Jan-2025
	Tags        []Tag
	Related     []Post
	Previous    []Post
	Next        []Post
}

type RSSFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel RSSChannel `xml:"channel"`
}

type RSSChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language"`
	Copyright   string    `xml:"copyright,omitempty"`
	Generator   string    `xml:"generator"`
	Items       []RSSItem `xml:"item"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Guid        string `xml:"guid"`
	Description string `xml:"description"`
	Author      string `xml:"author,omitempty"`
	PubDate     string `xml:"pubDate"`
}

type URLSet struct {
	XMLName xml.Name `xml:"urlset"`
	Xmlns   string   `xml:"xmlns,attr"`
	URLs    []URL    `xml:"url"`
}

type URL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

type PlainTextRenderer struct {
	buf bytes.Buffer
}

func loadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return &config, nil
}

func publish(md []byte) []byte {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return markdown.Render(doc, renderer)
}

/*
 * filePath = path to a post
 *
 * returns: frontMatter, content, text, err
 */
func parseFileWithHeaders(filePath string) (*FrontMatter, string, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to open file '%s': %w", filePath, err)
	}
	defer file.Close()

	var frontMatter FrontMatter
	var contentBuilder strings.Builder

	// Read the file and extract front matter and content
	var frontMatterBuilder strings.Builder
	inFrontMatter := false
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			inFrontMatter = !inFrontMatter
			continue
		}

		if inFrontMatter {
			frontMatterBuilder.WriteString(line + "\n")
		} else {
			contentBuilder.WriteString(line + "\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, "", "", fmt.Errorf("failed to read file '%s': %w", filePath, err)
	}

	// Parse front matter as YAML
	yamlContent := frontMatterBuilder.String()
	if err := yaml.Unmarshal([]byte(yamlContent), &frontMatter); err != nil {
		return nil, "", "", fmt.Errorf("failed to parse front matter: %w\nYAML:\n%s", err, yamlContent)
	}

	// Trim leading newline from content
	content := strings.TrimPrefix(contentBuilder.String(), "\n")
	return &frontMatter, content, ToPlainText(content), nil
}

func validateHeaders(frontMatter FrontMatter, filePath string) error {
	var errorMessages []string

	// Check for missing required headers
	if frontMatter.Title == "" {
		errorMessages = append(errorMessages, "missing a required header: title")
	}
	if frontMatter.Link == "" {
		errorMessages = append(errorMessages, "missing a required header: link")
	}
	if frontMatter.Published == "" {
		errorMessages = append(errorMessages, "missing a required header: published")
	}
	if frontMatter.Template == "" {
		errorMessages = append(errorMessages, "missing a required header: template")
	}
	if frontMatter.Description == "" {
		errorMessages = append(errorMessages, "missing a required header: description")
	}

	// Validate "status" field
	if _, valid := ValidPostStatuses[PostStatus(frontMatter.Status)]; !valid {
		errorMessages = append(errorMessages, fmt.Sprintf("Invalid value for status: %s", frontMatter.Status))
	}

	// Aggregate and return errors
	if len(errorMessages) > 0 {
		return fmt.Errorf("Post %s has the following issues:\n%s", filePath, strings.Join(errorMessages, "\n"))
	}

	return nil
}

/*
 * Sanity check: Validating user-provided link names
 */

func validateLinkName(link string) error {
	// Disallow reserved names
	if link == "." || link == ".." {
		return errors.New("invalid name: '.' and '..' are not allowed")
	}

	// Disallow path traversal patterns
	if strings.Contains(link, "..") {
		return errors.New("invalid name: path traversal patterns like '..' are not allowed")
	}

	// Disallow illegal characters
	illegalChars := regexp.MustCompile(`[<>:"/\\|?*\n\r\t]`)
	if illegalChars.MatchString(link) {
		return errors.New("invalid name: contains illegal characters (e.g., < > : \" / \\ | ? *)")
	}

	// Check for leading/trailing whitespace
	if strings.TrimSpace(link) != link {
		return errors.New("invalid name: leading or trailing whitespace is not allowed")
	}

	// Ensure the name is not empty and is of reasonable length
	if len(link) == 0 || len(link) > 255 {
		return errors.New("invalid name: must be between 1 and 255 characters long")
	}

	return nil
}

func reverse(posts []Post) []Post {
	reversed := make([]Post, len(posts))
	for i, post := range posts {
		reversed[len(posts)-1-i] = post
	}
	return reversed
}

func loadBadges(badgesDir string) map[string]template.HTML {
	badges := make(map[string]template.HTML)

	badgeFiles, err := os.ReadDir(badgesDir)
	if err != nil {
		log.Fatalf("Failed to read directory '%s': %v", badgesDir, err)
	}

	for _, badgeFile := range badgeFiles {
		if badgeFile.IsDir() {
			continue
		}
		badgePath := filepath.Join(badgesDir, badgeFile.Name())
		content, err := os.ReadFile(badgePath)
		if err != nil {
			log.Fatalf("Failed to read file: %s", err)
		}
		badges[badgeFile.Name()] = template.HTML(string(content))
	}

	return badges
}

/*
 * Helper code for transforming Markdown to text/plain for indexing purposes
 */

func (r *PlainTextRenderer) RenderNode(w io.Writer, node ast.Node, entering bool) ast.WalkStatus {
	switch n := node.(type) {
	case *ast.Text:
		r.buf.Write(n.Literal)
	case *ast.Link:
		if !entering {
			r.buf.WriteString(" (" + string(n.Destination) + ")")
		}
	case *ast.Heading:
		if !entering {
			r.buf.WriteString("\n")
		}
	case *ast.Paragraph:
		if !entering {
			r.buf.WriteString("\n\n")
		}
	case *ast.Code:
		r.buf.Write(n.Literal)
	case *ast.CodeBlock:
		r.buf.Write(n.Literal)
		r.buf.WriteString("\n")
	case *ast.ListItem:
		if entering {
			r.buf.WriteString("- ")
		}
	case *ast.HTMLBlock, *ast.HTMLSpan:
		return ast.SkipChildren
	}
	return ast.GoToNext
}

func (r *PlainTextRenderer) RenderHeader(w io.Writer, ast ast.Node) {}

func (r *PlainTextRenderer) RenderFooter(w io.Writer, ast ast.Node) {}

func ToPlainText(md string) string {
	parser := parser.NewWithExtensions(parser.CommonExtensions)
	doc := markdown.Parse([]byte(md), parser)
	renderer := &PlainTextRenderer{}
	markdown.Render(doc, renderer)
	return renderer.buf.String()
}

func processPosts(config Config) {
	/*
	 * Load badges into map: filename => SVG
	 */
	badges := loadBadges(config.BadgesDir)

	/*
	 * Fetch a list of all posts
	 */
	files, err := os.ReadDir(config.InputDir)
	if err != nil {
		log.Fatalf("Failed to read directory '%s': %v", config.InputDir, err)
	}

	/*
	 * Create output folder
	 */
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory '%s': %v", config.OutputDir, err)
	}

	/*
	 * Create tags folder
	 */
	tagsOutputDir := filepath.Join(config.OutputDir, "tags")
	if err := os.MkdirAll(tagsOutputDir, 0755); err != nil {
		log.Fatalf("Failed to create tags directory '%s': %v", config.OutputDir, err)
	}

	/*
	 * Create search folder
	 */
	if config.Search.Enabled {
		if err := os.MkdirAll(filepath.Join(config.OutputDir, config.Search.Dir), 0755); err != nil {
			log.Fatalf("Failed to create search directory '%s': %v", config.Search.Dir, err)
		}
	}

	/*
	 * List of all posts
	 */
	var posts []Post

	/*
	 * Map of one tag to many posts
	 */
	tagIndex := make(map[Tag][]Post)

	/*
	 * Timestamp
	 */
	now := time.Now().Format("January 2, 2006 at 3:04 PM")

	links := Links{
		Home:    buildRootLink(config),
		Atom:    buildAtomLink(config),
		RSS:     buildRSSLink(config),
		Tags:    buildTagsLink(config),
		Sitemap: buildSitemapLink(config),
		Rights:  buildRightsLink(config),
	}

	/*
	 * Pre-process all posts so we can show back/next
	 */

	postIndex := make(map[string]Post)
	var post Post
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		post = generatePost(config, file)

		/*
		 * Skip private posts
		 */
		status := PostStatus(post.FrontMatter.Status)
		if status == Private {
			fmt.Printf("📕 Post: %s [private] skipping...\n", post.FrontMatter.Link)
			continue
		}

		/*
		 * Check for duplicate links in posts
		 */
		if _, ok := postIndex[post.FrontMatter.Link]; ok {
			log.Fatalf("Duplicate link in post %v", post.FrontMatter.Link)
		} else {
			postIndex[post.FrontMatter.Link] = post
		}

		posts = append(posts, post)

		for _, tag := range post.Tags {
			tagIndex[tag] = append(tagIndex[tag], post)
		}
	}

	/*
	 * Check for duplicate links across pages
	 */
	for _, page := range config.Pages {
		if _, ok := postIndex[page.Link]; ok {
			log.Fatalf("Duplicate link in page %v", page.Link)
		}
	}

	/*
	 * Display new to old
	 */
	posts = reverse(posts)

	/*
	 * Convert each post from Markdown to HTML
	 */
	for i, post := range posts {
		templatePath := filepath.Join(config.TemplatesDir, post.FrontMatter.Template)
		tmpl, err := template.ParseFiles(templatePath, filepath.Join(config.TemplatesDir, "shared.html"))
		if err != nil {
			log.Fatalf("Failed to parse template '%s': %v", post.FrontMatter.Template, err)
		}
		labels := Labels{
			Title: post.FrontMatter.Title,
		}

		/*
		 * Collect all tags as we go
		 */
		var tags []Tag
		tags = append(tags, post.Tags...)

		/*
		 * Generate list of tag names
		 */
		tagNames := make([]string, len(tags))
		for i, tag := range tags {
			tagNames[i] = tag.TagName
		}

		unfurl := Unfurl{
			Title:       post.FrontMatter.Title,
			URL:         post.URL,
			Author:      post.FrontMatter.Author,
			Description: post.FrontMatter.Description,
			SiteName:    config.BlogName,
			Tags:        strings.Join(tagNames, ","),
			Locale:      config.Locale,
		}

		htmlContent := publish([]byte(post.HTML))

		/*
		 * Determine previous/next posts
		 */
		if i != len(posts)-1 {
			post.Previous = append(post.Previous, posts[i+1])
		}
		if i > 0 {
			post.Next = append(post.Next, posts[i-1])
		}

		// transform list of related posts by label to a `Related` struct
		var related []Post
		for _, link := range post.FrontMatter.Related {
			related = append(related, postIndex[link])
		}
		post.Related = related

		/*
		 * Template variables
		 */
		data := map[string]interface{}{
			"Config":    config,
			"Labels":    labels,
			"Unfurl":    unfurl,
			"Post":      post,
			"Content":   template.HTML(htmlContent),
			"Tags":      tags,
			"Version":   Version,
			"Now":       now,
			"Canonical": post.URL,
			"Links":     links,
			"Badges":    badges,
		}

		/*
		 * Write post to disk with folder name specified as `link` in metadata
		 */
		postDir := filepath.Join(config.OutputDir, post.FrontMatter.Link)
		if err := os.MkdirAll(postDir, 0755); err != nil {
			log.Fatalf("Failed to create directory '%s': %v", postDir, err)
		}

		outputFilePath := filepath.Join(postDir, "index.html")
		outputFile, err := os.Create(outputFilePath)
		if err != nil {
			log.Fatalf("Failed to create output file '%s': %v", outputFilePath, err)
		}
		defer outputFile.Close()

		if err := tmpl.Execute(outputFile, data); err != nil {
			log.Fatalf("Failed to execute template for file '%s': %v", outputFilePath, err)
		}

		fmt.Printf("📘 Post: \"%s\" by %s\n", post.FrontMatter.Link, post.FrontMatter.Author)
	}

	generateIndexHTML(config, posts, links, badges, now)
	generateTagsHTML(config, tagsOutputDir, tagIndex, links, badges, now)
	generateRSSFeed(config, posts)
	generateAtomFeed(config, posts)
	generateCustomPages(config, links, badges, now)
	generateSitemap(config, posts)
	if config.Search.Enabled {
		generateSluggoExport(config, posts)
		generateSearchHTML(config, links, badges, now)
	}
}

func generatePost(config Config, file fs.DirEntry) Post {
	filePath := filepath.Join(config.InputDir, file.Name())
	frontMatter, content, text, err := parseFileWithHeaders(filePath)

	if err != nil {
		log.Fatalf("Failed to process file '%s': %v", filePath, err)
	}

	if err := validateHeaders(*frontMatter, filePath); err != nil {
		log.Fatalf("Validation error for file '%s': %v", filePath, err)
	}

	if err := validateLinkName(frontMatter.Link); err != nil {
		log.Fatalf("Validation error for file '%s': %v", filePath, err)
	}

	// make tag structs: tag name, URL
	var tags []Tag
	for _, tag := range frontMatter.Tags {
		tag = strings.TrimSpace(tag)
		tags = append(tags, Tag{TagName: tag, URL: buildTagLink(config, tag)})
	}

	pubTime, err := time.Parse(time.RFC3339, frontMatter.Published)
	if err != nil {
		log.Fatalf("Error parsing date for %s: %v", filePath, err)
	}

	post := Post{
		FrontMatter: *frontMatter,
		URL:         buildPostLink(config, frontMatter.Link),
		HTML:        content,
		Text:        text,
		PubDate:     pubTime.Format("02-Jan-2006"),
		PubTime:     pubTime,
		Tags:        tags,
	}

	return post
}

func generateIndexHTML(config Config, posts []Post, links Links, badges map[string]template.HTML, now string) {
	tmpl, err := template.ParseFiles(config.IndexTemplatePath, filepath.Join(config.TemplatesDir, "shared.html"))
	if err != nil {
		log.Fatalf("Failed to parse index template '%s': %v", config.IndexTemplatePath, err)
	}

	indexFilePath := filepath.Join(config.OutputDir, "index.html")
	indexFile, err := os.Create(indexFilePath)
	if err != nil {
		log.Fatalf("Failed to create index file '%s': %v", indexFilePath, err)
	}
	defer indexFile.Close()

	labels := Labels{
		Title: config.BlogName,
	}

	url := buildRootLink(config)

	unfurl := Unfurl{
		Title:       config.BlogName,
		URL:         url,
		Description: config.Description,
		SiteName:    config.BlogName,
		Locale:      config.Locale,
	}

	data := map[string]interface{}{
		"Config":    config,
		"Labels":    labels,
		"Posts":     posts,
		"Version":   Version,
		"Now":       now,
		"Canonical": url,
		"Links":     links,
		"Unfurl":    unfurl,
		"Badges":    badges,
	}

	if err := tmpl.Execute(indexFile, data); err != nil {
		log.Fatalf("Failed to generate index.html: %v", err)
	}

	fmt.Printf("📙 Index: %s\n", indexFilePath)
}

func generateTagsHTML(config Config, tagsOutputDir string, tagIndex map[Tag][]Post, links Links, badges map[string]template.HTML, now string) {
	tmpl, err := template.ParseFiles(config.TagsIndexTemplatePath, filepath.Join(config.TemplatesDir, "shared.html"))
	if err != nil {
		log.Fatalf("Failed to parse tags index template '%s': %v", config.TagsIndexTemplatePath, err)
	}

	tagsIndexFilePath := filepath.Join(tagsOutputDir, "index.html")
	indexFile, err := os.Create(tagsIndexFilePath)
	if err != nil {
		log.Fatalf("Failed to create tags index file '%s': %v", tagsIndexFilePath, err)
	}
	defer indexFile.Close()

	labels := Labels{
		Title: config.BlogName + " Tags",
	}

	unfurl := Unfurl{
		Title:       config.BlogName,
		URL:         links.Tags,
		Description: config.BlogName + ": Tags",
		SiteName:    config.BlogName,
		Locale:      config.Locale,
	}

	data := map[string]interface{}{
		"Config":    config,
		"Labels":    labels,
		"Tags":      tagIndex,
		"Version":   Version,
		"Now":       now,
		"Canonical": links.Tags,
		"Links":     links,
		"Unfurl":    unfurl,
		"Badges":    badges,
	}

	if err := tmpl.Execute(indexFile, data); err != nil {
		log.Fatalf("Failed to generate tags index.html: %v", err)
	}
	fmt.Printf("📓 Tag Index: %s\n", tagsIndexFilePath)

	tagPageTemplate, err := template.ParseFiles(config.TagPageTemplatePath, filepath.Join(config.TemplatesDir, "shared.html"))
	if err != nil {
		log.Fatalf("Failed to parse tag page template '%s': %v", config.TagPageTemplatePath, err)
	}

	for tag, posts := range tagIndex {
		tagDir := filepath.Join(tagsOutputDir, tag.TagName)
		if err := os.MkdirAll(tagDir, 0755); err != nil {
			log.Fatalf("Failed to create directory for tag '%s': %v", tag.TagName, err)
		}

		tagFilePath := filepath.Join(tagDir, "index.html")
		tagFile, err := os.Create(tagFilePath)
		if err != nil {
			log.Fatalf("Failed to create tag file '%s': %v", tagFilePath, err)
		}
		defer tagFile.Close()

		labels := Labels{
			Title: config.BlogName + " Tags",
		}

		url := buildTagLink(config, tag.TagName)

		unfurl := Unfurl{
			Title:       config.BlogName,
			URL:         url,
			Description: config.BlogName + ": Posts tagged " + tag.TagName,
			SiteName:    config.BlogName,
			Locale:      config.Locale,
		}

		data := map[string]interface{}{
			"Config":    config,
			"Labels":    labels,
			"Key":       tag.TagName,
			"Value":     posts,
			"Version":   Version,
			"Now":       now,
			"Canonical": url,
			"Links":     links,
			"Unfurl":    unfurl,
			"Badges":    badges,
		}

		if err := tagPageTemplate.Execute(tagFile, data); err != nil {
			log.Fatalf("Failed to generate tag page '%s': %v", tagFilePath, err)
		}
		fmt.Printf("📓 Tag: %s\n", tag.TagName)
	}
}

func generateCustomPages(config Config, links Links, badges map[string]template.HTML, now string) {
	for _, page := range config.Pages {
		templatePath := filepath.Join(config.TemplatesDir, page.Template)
		tmpl, err := template.ParseFiles(templatePath, filepath.Join(config.TemplatesDir, "shared.html"))
		if err != nil {
			log.Fatalf("Failed to parse template '%s': %v", templatePath, err)
		}
		labels := Labels{
			Title: page.Title,
		}

		url := buildCustomPageLink(config, page)
		unfurl := Unfurl{
			Title:       config.BlogName,
			URL:         url,
			Description: page.Title,
			SiteName:    config.BlogName,
			Locale:      config.Locale,
		}

		data := map[string]interface{}{
			"Config":    config,
			"Labels":    labels,
			"Version":   Version,
			"Now":       now,
			"Canonical": url,
			"Links":     links,
			"Unfurl":    unfurl,
			"Badges":    badges,
		}

		customPageDir := filepath.Join(config.OutputDir, page.Link)
		if err := os.MkdirAll(customPageDir, 0755); err != nil {
			log.Fatalf("Failed to create directory '%s': %v", customPageDir, err)
		}
		customPagePath := filepath.Join(customPageDir, "index.html")

		outputFile, err := os.Create(customPagePath)
		if err != nil {
			log.Fatalf("Failed to create output file '%s': %v", customPagePath, err)
		}
		defer outputFile.Close()

		if err := tmpl.Execute(outputFile, data); err != nil {
			log.Fatalf("Failed to execute template for file '%s': %v", customPagePath, err)
		}

		fmt.Printf("📘 Page: %s\n", page.Title)
	}
}

/*
 * Sitemap
 */

func generateSitemap(config Config, posts []Post) {
	var urls []URL

	// Home page
	urls = append(urls, URL{
		Loc:        buildRootLink(config),
		LastMod:    time.Now().Format("2006-01-02"),
		ChangeFreq: "daily",
		Priority:   "1.0",
	})

	// Tags page
	urls = append(urls, URL{
		Loc:        buildTagsLink(config),
		LastMod:    time.Now().Format("2006-01-02"),
		ChangeFreq: "weekly",
		Priority:   "0.8",
	})

	// RSS page
	urls = append(urls, URL{
		Loc:      buildRSSLink(config),
		LastMod:  time.Now().Format("2006-01-02"),
		Priority: "0.7",
	})

	// Static pages from Config.Pages
	for _, page := range config.Pages {
		urls = append(urls, URL{
			Loc:        buildCustomPageLink(config, page),
			LastMod:    time.Now().Format("2006-01-02"),
			ChangeFreq: "monthly",
			Priority:   "0.5",
		})
	}

	// Posts
	for _, post := range posts {
		urls = append(urls, URL{
			Loc:        buildPostLink(config, post.FrontMatter.Link),
			LastMod:    post.PubTime.Format(time.RFC3339), // ISO 8601
			ChangeFreq: "weekly",
			Priority:   "0.9",
		})
	}

	sitemap := URLSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	outputFilePath := filepath.Join(config.OutputDir, "sitemap.xml")
	file, err := os.Create(outputFilePath)
	if err != nil {
		fmt.Printf("Error creating sitemap file: %v\n", err)
		return
	}
	defer file.Close()

	encoder := xml.NewEncoder(file)
	encoder.Indent("", "  ")
	file.WriteString(xml.Header)
	if err := encoder.Encode(sitemap); err != nil {
		fmt.Printf("Error writing sitemap to file: %v\n", err)
		return
	}

	fmt.Printf("📔 Sitemap %s\n", outputFilePath)
}

/*
 * Sluggo version 1 structures
 */

type Corpus struct {
	Name      string     // Name of corpus
	URL       string     // URL e.g. https://harrison.blog
	Created   int64      // Create date in UNIX time since epoch
	Version   int        // Version of JSON input file
	Documents []Document // List of documents
}

type Document struct {
	ID          string              // Document ID, typically a URL
	Title       string              // Docuemnt title e.g. "Hello World"
	Description string              // Brief description, first sentence of document or so
	Text        string              // Entire document
	Attributes  map[string][]string // Arbitrary attributes e.g. {"author": ["Harrison"]}
	Hints       []string            // Hints provide best matches for search
}

func convertTagsToStrings(tags []Tag) []string {
	tagStrings := make([]string, len(tags))
	for i, tag := range tags {
		tagStrings[i] = tag.TagName
	}
	return tagStrings
}

/*
 * Optionally generate Sluggo (search engine) export
 */

func generateSluggoExport(config Config, posts []Post) {
	file, err := os.Create(config.Search.Path)
	if err != nil {
		fmt.Printf("Error creating sluggo export file '%s': %v\n", config.Search.Path, err)
		return
	}
	payload := Corpus{
		Name:    config.BlogName,
		URL:     config.URL,
		Created: time.Now().Unix(),
		Version: 1,
	}
	for _, post := range posts {
		doc := Document{
			ID:          post.URL,
			Title:       post.FrontMatter.Title,
			Description: post.FrontMatter.Description,
			Text:        post.Text,
			Attributes:  map[string][]string{"author": {post.FrontMatter.Author}, "tags": convertTagsToStrings(post.Tags)},
			Hints:       []string{},
		}
		payload.Documents = append(payload.Documents, doc)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(payload); err != nil {
		fmt.Printf("Error writing sluggo export to file: %v\n", err)
		return
	}

	fmt.Printf("📔 Sluggo export: %s\n", config.Search.Path)
}

func generateSearchHTML(config Config, links Links, badges map[string]template.HTML, now string) {
	templatePath := filepath.Join(config.TemplatesDir, "search.html")
	tmpl, err := template.ParseFiles(templatePath, filepath.Join(config.TemplatesDir, "shared.html"))
	if err != nil {
		log.Fatalf("Failed to parse template '%s': %v", templatePath, err)
	}
	searchPath := filepath.Join(config.OutputDir, config.Search.Dir, "index.html")
	searchFile, err := os.Create(searchPath)
	if err != nil {
		log.Fatalf("Failed to create search file '%s': %v", searchPath, err)
	}
	defer searchFile.Close()
	data := map[string]interface{}{
		"Config": config,
		"Links":  links,
		"Badges": badges,
		"Now":    now,
	}
	if err := tmpl.Execute(searchFile, data); err != nil {
		log.Fatalf("Failed to generate tags index.html: %v", err)
	}
	fmt.Printf("🔍 Search template written")
}

/*
 * For functions that build paths or URLs, we check if an optional BasePath is
 * set. This allows us to serve documents from example.com/blog rather than
 * the root of example.com.
 */

func buildPostLink(config Config, link string) string {
	if config.BasePath != "" {
		return fmt.Sprintf("%s/%s/%s/", config.URL, config.BasePath, link)
	}
	return fmt.Sprintf("%s/%s/", config.URL, link)
}

func buildRootLink(config Config) string {
	if config.BasePath != "" {
		return fmt.Sprintf("%s/%s/", config.URL, config.BasePath)
	}
	return fmt.Sprintf("%s/", config.URL)
}

func buildTagLink(config Config, tag string) string {
	if config.BasePath != "" {
		return fmt.Sprintf("%s/%s/tags/%s/", config.URL, config.BasePath, tag)
	}
	return fmt.Sprintf("%s/tags/%s/", config.URL, tag)
}

func buildTagsLink(config Config) string {
	if config.BasePath != "" {
		return fmt.Sprintf("%s/%s/tags/", config.URL, config.BasePath)
	}
	return fmt.Sprintf("%s/tags/", config.URL)
}

func buildRSSLink(config Config) string {
	if config.BasePath != "" {
		return fmt.Sprintf("%s/%s/rss.xml", config.URL, config.BasePath)
	}
	return fmt.Sprintf("%s/rss.xml", config.URL)
}

func buildAtomLink(config Config) string {
	if config.BasePath != "" {
		return fmt.Sprintf("%s/%s/atom.xml", config.URL, config.BasePath)
	}
	return fmt.Sprintf("%s/atom.xml", config.URL)
}

func buildCustomPageLink(config Config, page Page) string {
	if config.BasePath != "" {
		return fmt.Sprintf("%s/%s/%s/", config.URL, config.BasePath, page.Link)
	}
	return fmt.Sprintf("%s/%s", config.URL, page.Link)
}

func buildSitemapLink(config Config) string {
	if config.BasePath != "" {
		return fmt.Sprintf("%s/%s/%s", config.URL, config.BasePath, "sitemap.xml")
	}
	return fmt.Sprintf("%s/%s", config.URL, "sitemap.xml")
}

func buildRightsLink(config Config) string {
	if config.BasePath != "" {
		return fmt.Sprintf("%s/%s/%s", config.URL, config.BasePath, "sitemap.xml")
	}
	return fmt.Sprintf("%s/%s", config.URL, "rights/")
}

/*
 * RSS 2.0
 */

func generateRSSFeed(config Config, posts []Post) error {
	items := make([]RSSItem, len(posts))
	for i, post := range posts {
		items[i] = RSSItem{
			Title:       post.FrontMatter.Title,
			Link:        post.URL,
			Guid:        post.URL,
			Description: post.FrontMatter.Description,
			Author:      post.FrontMatter.Author,
			PubDate:     post.PubTime.Format(time.RFC1123Z), // RFC 1123
		}
	}

	rss := RSSFeed{
		Version: "2.0",
		Channel: RSSChannel{
			Title:       config.BlogName,
			Link:        buildRootLink(config),
			Description: fmt.Sprintf("Latest posts from %s", config.BlogName),
			Language:    config.Language,
			Copyright:   config.Rights,
			Generator:   "Draft/" + Version,
			Items:       items,
		},
	}

	outputPath := fmt.Sprintf("%s/rss.xml", config.OutputDir)
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create RSS feed file: %w", err)
	}
	defer file.Close()

	encoder := xml.NewEncoder(file)
	encoder.Indent("", "  ")
	if err := encoder.Encode(rss); err != nil {
		return fmt.Errorf("failed to encode RSS feed: %w", err)
	}

	fmt.Printf("📔 RSS: %s\n", outputPath)
	return nil
}

/*
 * Atom
 */

func generateAtomFeed(config Config, posts []Post) error {
	type AtomLink struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr,omitempty"`
		Type string `xml:"type,attr,omitempty"`
	}

	type AtomAuthor struct {
		Name  string `xml:"name"`
		Email string `xml:"email,omitempty"`
	}

	type AtomEntry struct {
		Title     string     `xml:"title"`
		Link      []AtomLink `xml:"link"`
		Id        string     `xml:"id"`
		Published string     `xml:"published,omitempty"`
		Updated   string     `xml:"updated"`
		Summary   string     `xml:"summary"`
		Content   string     `xml:"content,omitempty"`
		Author    AtomAuthor `xml:"author"`
	}

	type AtomFeed struct {
		XMLName  xml.Name    `xml:"feed"`
		Xmlns    string      `xml:"xmlns,attr"`
		Title    string      `xml:"title"`
		Subtitle string      `xml:"subtitle"`
		Link     []AtomLink  `xml:"link"`
		Id       string      `xml:"id"`
		Updated  string      `xml:"updated"`
		Author   AtomAuthor  `xml:"author"`
		Entries  []AtomEntry `xml:"entry"`
	}

	entries := make([]AtomEntry, len(posts))
	for i, post := range posts {
		entries[i] = AtomEntry{
			Title: post.FrontMatter.Title,
			Link: []AtomLink{
				{Href: post.URL},
			},
			Id:        post.URL,
			Published: post.PubTime.Format(time.RFC3339),
			Updated:   post.PubTime.Format(time.RFC3339),
			Summary:   post.FrontMatter.Description,
			Author:    AtomAuthor{Name: post.FrontMatter.Author, Email: post.FrontMatter.Email},
		}
	}

	atom := AtomFeed{
		Xmlns:    "http://www.w3.org/2005/Atom",
		Title:    config.BlogName,
		Subtitle: fmt.Sprintf("Latest posts from %s", config.BlogName),
		Link: []AtomLink{
			{Href: buildAtomLink(config), Rel: "self"},
			{Href: buildRootLink(config)},
		},
		Id:      buildRootLink(config),
		Updated: time.Now().Format(time.RFC3339),
		Author:  AtomAuthor{Name: config.BlogName, Email: config.Email},
		Entries: entries,
	}

	outputPath := fmt.Sprintf("%s/atom.xml", config.OutputDir)
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create Atom feed file: %w", err)
	}
	defer file.Close()

	encoder := xml.NewEncoder(file)
	encoder.Indent("", "  ")
	if err := encoder.Encode(atom); err != nil {
		return fmt.Errorf("failed to encode Atom feed: %w", err)
	}

	fmt.Printf("⚛️  Atom: %s\n", outputPath)
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: draft [config.yaml]")
		os.Exit(1)
	}

	fmt.Printf("📗 Draft version %s (%s)\n", Version, BuildDate)
	fmt.Printf("🤓 https://github.com/harrisonpage/draft\n")

	configPath := os.Args[1]
	config, err := loadConfig(configPath)
	if err != nil {
		if configPath == "--help" || configPath == "-h" || configPath == "help" {
			fmt.Printf("🆘 See also: https://harrison.blog/announcing-draft/\n")
		} else {
			fmt.Printf("Error loading configuration: %v\n", err)
		}
		os.Exit(1)
	}

	processPosts(*config)
}
