package main

import (
	"bufio"
	"encoding/xml"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
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

/*
 * List of post headers and whether they are required
 */
var Headers = map[string]bool{
	"title":       true,
	"link":        true,
	"published":   true,
	"template":    true,
	"description": true,
	"tags":        false,
	"favicon":     false,
	"author":      false,
	"image":       false,
	"status":      false,
}

/*
 * Fields in config.yaml
 */
type Config struct {
	InputDir              string   `yaml:"input_dir"`
	TemplatesDir          string   `yaml:"templates_dir"`
	OutputDir             string   `yaml:"output_dir"`
	BadgesDir             string   `yaml:"badges_dir"`
	IndexTemplatePath     string   `yaml:"index_template_path"`
	TagsIndexTemplatePath string   `yaml:"tags_index_template_path"`
	TagPageTemplatePath   string   `yaml:"tag_page_template_path"`
	Author                string   `yaml:"author"`
	BlogName              string   `yaml:"blog_name"`
	Description           string   `yaml:"description"`
	Copyright             string   `yaml:"copyright"`
	Language              string   `yaml:"language"`
	Locale                string   `yaml:"locale"`
	Lang                  string   `yaml:"lang"`
	BackLabel             string   `yaml:"back_label"`
	CSSFiles              []string `yaml:"css_files"`
	JSFiles               []string `yaml:"js_files"`
	Pages                 []Page   `yaml:"pages"`
	URL                   string   `yaml:"url"`
	BasePath              string   `yaml:"base_path"`
	Badges                []Badge  `yaml:"badges"`
	FediverseCreator      string   `yaml:"fediverse_creator"`
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
	Sitemap string
}

type Tag struct {
	TagName string
	URL     string
}

type Post struct {
	Title       string
	Author      string
	Link        string
	URL         string
	Template    string
	Content     string
	Published   string    // ISO 8601 AKA time.RFC3339 e.g. 2025-01-15T06:29:00-08:00
	PubTime     time.Time // parsed version of Published date
	PubDate     string    // 15-Jan-2025
	Description string
	Tags        []Tag
	Image       string
	Favicon     string
	Status      string
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

func parseFileWithHeaders(filePath string) (map[string]string, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open file '%s': %w", filePath, err)
	}
	defer file.Close()

	headers := make(map[string]string)
	var contentBuilder strings.Builder
	yip := false
	i := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		i += 1
		line := scanner.Text()
		if line == "---" {
			if i == 1 {
				// skip first delimiter
				continue
			} else {
				// end of header metadata
				yip = true
			}
		}

		if yip {
			contentBuilder.WriteString(line + "\n")
		} else {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				headers[key] = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("failed to read file '%s': %w", filePath, err)
	}

	return headers, contentBuilder.String(), nil
}

func validateHeaders(headers map[string]string, knownHeaders map[string]bool, filePath string) error {
	var errorMessages []string

	// Check for missing required headers in sorted order
	var requiredHeaders []string
	for header, required := range knownHeaders {
		if required {
			requiredHeaders = append(requiredHeaders, header)
		}
	}
	sort.Strings(requiredHeaders)

	for _, header := range requiredHeaders {
		if _, exists := headers[header]; !exists {
			errorMessages = append(errorMessages, fmt.Sprintf("missing a required header: %s", header))
		}
	}

	// Check for unknown headers (order does not matter here)
	for header := range headers {
		if _, exists := knownHeaders[header]; !exists {
			errorMessages = append(errorMessages, fmt.Sprintf("contains unknown header: %s", header))
		}
	}

	if _, valid := ValidPostStatuses[PostStatus(headers["status"])]; !valid {
		errorMessages = append(errorMessages, fmt.Sprintf("Invalid value for status: %s", headers["status"]))
	}

	if len(errorMessages) > 0 {
		return fmt.Errorf("Post %s has the following issues:\n%s", filePath, strings.Join(errorMessages, "\n"))
	}

	return nil
}

/*
 * Sanity check: Validating user-provided link names even if the user is, like,
 * super smart and never makes mistakes.
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

func processMarkdownFiles(config Config) {
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
		RSS:     buildRSSLink(config),
		Tags:    buildTagsLink(config),
		Sitemap: buildSitemapLink(config),
	}

	namespace := make(map[string]bool)

	/*
	 * Pre-process all posts so we can show back/next
	 */
	var post Post
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		post = generatePost(config, file)

		/*
		 * Skip private posts
		 */
		status := PostStatus(post.Status)
		if status == Private {
			fmt.Printf("📕 Post: %s [private] skipping...\n", post.Link)
			continue
		}

		/*
		 * Check for duplicate links in posts
		 */
		if _, ok := namespace[post.Link]; ok {
			log.Fatalf("Duplicate link in post %v", post.Link)
		} else {
			namespace[post.Link] = true
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
		if _, ok := namespace[page.Link]; ok {
			log.Fatalf("Duplicate link in page %v", page.Link)
		} else {
			namespace[page.Link] = true
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
		templatePath := filepath.Join(config.TemplatesDir, post.Template)
		tmpl, err := template.ParseFiles(templatePath, filepath.Join(config.TemplatesDir, "shared.html"))
		if err != nil {
			log.Fatalf("Failed to parse template '%s': %v", post.Template, err)
		}
		labels := Labels{
			Title: post.Title,
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
			Title:       post.Title,
			URL:         post.URL,
			Author:      post.Author,
			Description: post.Description,
			SiteName:    config.BlogName,
			Tags:        strings.Join(tagNames, ","),
			Locale:      config.Locale,
		}

		htmlContent := publish([]byte(post.Content))

		/*
		 * Determine previous/next posts
		 */
		var previousPost Post
		if i != len(posts)-1 {
			previousPost = posts[i+1]
		} else {
			previousPost = Post{}
		}
		var nextPost Post
		if i > 0 {
			nextPost = posts[i-1]
		} else {
			nextPost = Post{}
		}

		/*
		 * Template variables
		 */
		data := map[string]interface{}{
			"Config":       config,
			"Labels":       labels,
			"Unfurl":       unfurl,
			"Post":         post,
			"Content":      template.HTML(htmlContent),
			"Tags":         tags,
			"Version":      Version,
			"Now":          now,
			"Canonical":    post.URL,
			"Links":        links,
			"Badges":       badges,
			"PreviousPost": previousPost,
			"NextPost":     nextPost,
		}

		/*
		 * Write post to disk with folder name specified as `link` in metadata
		 */
		postDir := filepath.Join(config.OutputDir, post.Link)
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

		fmt.Printf("📘 Post: %s\n", post.Link)
	}

	generateIndexHTML(config, posts, links, badges, now)
	generateTagsHTML(config, tagsOutputDir, tagIndex, links, badges, now)
	generateRSSFeed(config, posts)
	generateCustomPages(config, links, badges, now)
	generateSitemap(config, posts)
}

func generatePost(config Config, file fs.DirEntry) Post {
	filePath := filepath.Join(config.InputDir, file.Name())
	headers, content, err := parseFileWithHeaders(filePath)
	if err != nil {
		log.Fatalf("Failed to process file '%s': %v", filePath, err)
	}

	if err := validateHeaders(headers, Headers, filePath); err != nil {
		log.Fatalf("Validation error for file '%s': %v", filePath, err)
	}

	if err := validateLinkName(headers["link"]); err != nil {
		log.Fatalf("Validation error for file '%s': %v", filePath, err)
	}

	tagStrings := strings.Split(headers["tags"], ",")
	var tags []Tag
	for _, tag := range tagStrings {
		tag = strings.TrimSpace(tag)
		tags = append(tags, Tag{TagName: tag, URL: buildTagLink(config, tag)})
	}

	pubTime, err := time.Parse(time.RFC3339, headers["published"])
	if err != nil {
		log.Fatalf("Error parsing date for %s: %v", filePath, err)
	}

	post := Post{
		Title:       headers["title"],
		Link:        headers["link"],
		URL:         buildPostLink(config, headers["link"]),
		Content:     content,
		Template:    headers["template"],
		Published:   headers["published"],
		PubDate:     pubTime.Format("02-Jan-2006"),
		PubTime:     pubTime,
		Description: headers["description"],
		Tags:        tags,
		Image:       headers["image"],
		Favicon:     headers["favicon"],
		Status:      headers["status"],
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
			Loc:        buildPostLink(config, post.Link),
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

/*
 * RSS 2.0
 */

func generateRSSFeed(config Config, posts []Post) error {
	items := make([]RSSItem, len(posts))
	for i, post := range posts {
		items[i] = RSSItem{
			Title:       post.Title,
			Link:        post.URL,
			Guid:        post.URL,
			Description: post.Description,
			Author:      post.Author,
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
			Copyright:   config.Copyright,
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

	processMarkdownFiles(*config)
}
