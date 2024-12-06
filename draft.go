package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"gopkg.in/yaml.v3"
)

var Version string
var BuildDate string

/*
 * config.yaml
 */
type Config struct {
	InputDir              string   `yaml:"input_dir"`
	TemplatesDir          string   `yaml:"templates_dir"`
	OutputDir             string   `yaml:"output_dir"`
	IndexTemplatePath     string   `yaml:"index_template_path"`
	TagsIndexTemplatePath string   `yaml:"tags_index_template_path"`
	TagPageTemplatePath   string   `yaml:"tag_page_template_path"`
	Author                string   `yaml:"author"`
	BlogName              string   `yaml:"blog_name"`
	Description           string   `yaml:"description"`
	Copyright             string   `yaml:"copyright"`
	Language              string   `yaml:"language"`
	BackLabel             string   `yaml:"back_label"`
	CSSFiles              []string `yaml:"css_files"`
	JSFiles               []string `yaml:"js_files"`
	Pages                 []Page   `yaml:"pages"`
	URL                   string   `yaml:"url"`
	BasePath              string   `yaml:"base_path"`
}

type Page struct {
	Template string
	Title    string
	Path     string
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
}

type Links struct {
	Home string
	Tags string
	RSS  string
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
	Published   string
	Description string
	Tags        []Tag
	Image       string
	Favicon     string
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
	Items       []RSSItem `xml:"item"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Author      string `xml:"author,omitempty"`
	PubDate     string `xml:"pubDate"`
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
	inContent := false

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "==cut here==" {
			inContent = true
			continue
		}

		if inContent {
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

func validateHeaders(headers map[string]string, required []string) error {
	for _, key := range required {
		if _, ok := headers[key]; !ok {
			return fmt.Errorf("missing required header: %s", key)
		}
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

func processMarkdownFiles(config Config) {
	files, err := ioutil.ReadDir(config.InputDir)
	if err != nil {
		log.Fatalf("Failed to read directory '%s': %v", config.InputDir, err)
	}

	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory '%s': %v", config.OutputDir, err)
	}
	tagsOutputDir := filepath.Join(config.OutputDir, "tags")
	if err := os.MkdirAll(tagsOutputDir, 0755); err != nil {
		log.Fatalf("Failed to create tags directory '%s': %v", config.OutputDir, err)
	}

	var posts []Post
	tagIndex := make(map[Tag][]Post)
	now := time.Now().Format("January 2, 2006 at 3:04 PM")

	links := Links{
		RSS:  buildRSSLink(config),
		Tags: buildTagsLink(config),
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(config.InputDir, file.Name())
		headers, content, err := parseFileWithHeaders(filePath)
		if err != nil {
			log.Fatalf("Failed to process file '%s': %v", filePath, err)
		}

		requiredHeaders := []string{"title", "link", "published", "template", "description", "tags", "image", "favicon"}
		if err := validateHeaders(headers, requiredHeaders); err != nil {
			log.Fatalf("Validation error for file '%s': %v", filePath, err)
		}

		tagStrings := strings.Split(headers["tags"], ",")
		var tags []Tag
		var tagsRaw []string
		for _, tag := range tagStrings {
			tag = strings.TrimSpace(tag)
			tags = append(tags, Tag{TagName: tag, URL: buildTagLink(config, tag)})
			tagsRaw = append(tagsRaw, tag)
		}

		labels := Labels{
			Title: headers["title"],
		}

		url := buildPostLink(config, headers["link"])

		unfurl := Unfurl{
			Title:       headers["title"],
			URL:         url,
			Author:      headers["author"],
			Description: headers["description"],
			SiteName:    config.BlogName,
			Tags:        strings.Join(tagsRaw, ","),
		}

		post := Post{
			Title:       headers["title"],
			Link:        headers["link"],
			URL:         url,
			Published:   headers["published"],
			Description: headers["description"],
			Tags:        tags,
			Image:       headers["image"],
			Favicon:     headers["favicon"],
		}
		posts = append(posts, post)

		for _, tag := range tags {
			tagIndex[tag] = append(tagIndex[tag], post)
		}

		templateFile := headers["template"]
		templatePath := filepath.Join(config.TemplatesDir, templateFile)
		tmpl, err := template.ParseFiles(templatePath, filepath.Join(config.TemplatesDir, "shared.html"))
		if err != nil {
			log.Fatalf("Failed to parse template '%s': %v", templateFile, err)
		}

		htmlContent := publish([]byte(content))

		data := map[string]interface{}{
			"Config":    config,
			"Labels":    labels,
			"Unfurl":    unfurl,
			"Post":      post,
			"Content":   template.HTML(htmlContent),
			"Tags":      tags,
			"Version":   Version,
			"Now":       now,
			"Canonical": url,
			"Links":     links,
		}

		postDir := filepath.Join(config.OutputDir, headers["link"])
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
			log.Fatalf("Failed to execute template for file '%s': %v", filePath, err)
		}

		fmt.Printf("📘 Post: %s\n", headers["link"])
	}

	generateIndexHTML(config, posts, links, now)
	generateTagsHTML(config, tagsOutputDir, tagIndex, links, now)
	generateRSSFeed(config, posts)
	generateCustomPages(config, links, now)
}

func generateIndexHTML(config Config, posts []Post, links Links, now string) {
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
	}

	data := map[string]interface{}{
		"Config":    config,
		"Labels":    labels,
		"Posts":     reverse(posts),
		"Version":   Version,
		"Now":       now,
		"Canonical": url,
		"Links":     links,
		"Unfurl":    unfurl,
	}

	if err := tmpl.Execute(indexFile, data); err != nil {
		log.Fatalf("Failed to generate index.html: %v", err)
	}

	fmt.Printf("📙 Index: %s\n", indexFilePath)
}

func generateTagsHTML(config Config, tagsOutputDir string, tagIndex map[Tag][]Post, links Links, now string) {
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
		}

		if err := tagPageTemplate.Execute(tagFile, data); err != nil {
			log.Fatalf("Failed to generate tag page '%s': %v", tagFilePath, err)
		}
		fmt.Printf("📕 Tag: %s\n", tag.TagName)
	}
}

func generateCustomPages(config Config, links Links, now string) {
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
		}

		data := map[string]interface{}{
			"Config":    config,
			"Labels":    labels,
			"Version":   Version,
			"Now":       now,
			"Canonical": url,
			"Links":     links,
			"Unfurl":    unfurl,
		}

		customPageDir := filepath.Join(config.OutputDir, page.Path)
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
		return fmt.Sprintf("%s/%s/%s/", config.URL, config.BasePath, page.Path)
	}
	return fmt.Sprintf("%s/%s", config.URL, page.Path)
}

func generateRSSFeed(config Config, posts []Post) error {
	items := make([]RSSItem, len(posts))
	for i, post := range posts {
		items[i] = RSSItem{
			Title:       post.Title,
			Link:        post.URL,
			Description: post.Description,
			Author:      post.Author,
			PubDate:     post.Published, // .Format(time.RFC1123), // Format for RSS
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

	configPath := os.Args[1]
	config, err := loadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📗 Draft version %s (%s)\n", Version, BuildDate)
	fmt.Printf("🤓 https://github.com/harrisonpage/draft\n")
	processMarkdownFiles(*config)
}
