package main

import (
	"bufio"
	"encoding/xml"
	"flag"
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
)

var Version string

type Config struct {
	InputDir              string
	TemplatesDir          string
	OutputDir             string
	IndexTemplatePath     string
	TagsIndexTemplatePath string
	TagPageTemplatePath   string
	Author                string
	BlogName              string
	Copyright             string
	Language              string
	CSSFiles              []string
	JSFiles               []string
	URL                   string
}

type PostMetadata struct {
	Title       string
	Author      string
	Link        string
	Published   string
	Description string
	Tags        []string
	Image       string
	Favicon     string
}

// RSSFeed represents the structure of an RSS feed.
type RSSFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel RSSChannel `xml:"channel"`
}

// RSSChannel represents the channel element in the RSS feed.
type RSSChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language"`
	Copyright   string    `xml:"copyright,omitempty"`
	Items       []RSSItem `xml:"item"`
}

// RSSItem represents an individual item in the RSS feed.
type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Author      string `xml:"author,omitempty"`
	PubDate     string `xml:"pubDate"`
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

func reverse(posts []PostMetadata) []PostMetadata {
	reversed := make([]PostMetadata, len(posts))
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

	var posts []PostMetadata
	tagIndex := make(map[string][]PostMetadata)
	now := time.Now().Format("January 2, 2006 at 3:04 PM")

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

		tags := strings.Split(headers["tags"], ",")
		for i, tag := range tags {
			tags[i] = strings.TrimSpace(tag)
		}

		post := PostMetadata{
			Title:       headers["title"],
			Link:        headers["link"],
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
		tmpl, err := template.ParseFiles(templatePath)
		if err != nil {
			log.Fatalf("Failed to parse template '%s': %v", templateFile, err)
		}

		htmlContent := publish([]byte(content))

		data := map[string]interface{}{
			"Headers":   headers,
			"Content":   template.HTML(htmlContent),
			"Tags":      tags,
			"Version":   Version,
			"Author":    config.Author,
			"BlogName":  config.BlogName,
			"Copyright": config.Copyright,
			"CSSFiles":  config.CSSFiles,
			"JSFiles":   config.JSFiles,
			"Now":       now,
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

		fmt.Printf("📘 %s\n", headers["link"])
	}

	generateIndexHTML(config, posts, now)
	generateTagsHTML(config, tagsOutputDir, tagIndex, now)
	GenerateRSSFeed(config, posts)
}

func generateIndexHTML(config Config, posts []PostMetadata, now string) {
	tmpl, err := template.ParseFiles(config.IndexTemplatePath)
	if err != nil {
		log.Fatalf("Failed to parse index template '%s': %v", config.IndexTemplatePath, err)
	}

	indexFilePath := filepath.Join(config.OutputDir, "index.html")
	indexFile, err := os.Create(indexFilePath)
	if err != nil {
		log.Fatalf("Failed to create index file '%s': %v", indexFilePath, err)
	}
	defer indexFile.Close()

	data := map[string]interface{}{
		"Posts":     reverse(posts),
		"Version":   Version,
		"Author":    config.Author,
		"BlogName":  config.BlogName,
		"Copyright": config.Copyright,
		"CSSFiles":  config.CSSFiles,
		"JSFiles":   config.JSFiles,
		"Now":       now,
	}

	if err := tmpl.Execute(indexFile, data); err != nil {
		log.Fatalf("Failed to generate index.html: %v", err)
	}

	fmt.Printf("📙 %s\n", indexFilePath)
}

func generateTagsHTML(config Config, tagsOutputDir string, tagIndex map[string][]PostMetadata, now string) {
	tmpl, err := template.ParseFiles(config.TagsIndexTemplatePath)
	if err != nil {
		log.Fatalf("Failed to parse tags index template '%s': %v", config.TagsIndexTemplatePath, err)
	}

	tagsIndexFilePath := filepath.Join(tagsOutputDir, "index.html")
	indexFile, err := os.Create(tagsIndexFilePath)
	if err != nil {
		log.Fatalf("Failed to create tags index file '%s': %v", tagsIndexFilePath, err)
	}
	defer indexFile.Close()

	data := map[string]interface{}{
		"Tags":      tagIndex,
		"Version":   Version,
		"Author":    config.Author,
		"Copyright": config.Copyright,
		"CSSFiles":  config.CSSFiles,
		"JSFiles":   config.JSFiles,
		"Now":       now,
	}

	if err := tmpl.Execute(indexFile, data); err != nil {
		log.Fatalf("Failed to generate tags index.html: %v", err)
	}
	fmt.Printf("📓 %s\n", tagsIndexFilePath)

	tagPageTemplate, err := template.ParseFiles(config.TagPageTemplatePath)
	if err != nil {
		log.Fatalf("Failed to parse tag page template '%s': %v", config.TagPageTemplatePath, err)
	}

	for tag, posts := range tagIndex {
		tagDir := filepath.Join(tagsOutputDir, tag)
		if err := os.MkdirAll(tagDir, 0755); err != nil {
			log.Fatalf("Failed to create directory for tag '%s': %v", tag, err)
		}

		tagFilePath := filepath.Join(tagDir, "index.html")
		tagFile, err := os.Create(tagFilePath)
		if err != nil {
			log.Fatalf("Failed to create tag file '%s': %v", tagFilePath, err)
		}
		defer tagFile.Close()

		data := map[string]interface{}{
			"Key":       tag,
			"Value":     posts,
			"Version":   Version,
			"Author":    config.Author,
			"Copyright": config.Copyright,
			"CSSFiles":  config.CSSFiles,
			"JSFiles":   config.JSFiles,
			"Now":       now,
		}

		if err := tagPageTemplate.Execute(tagFile, data); err != nil {
			log.Fatalf("Failed to generate tag page '%s': %v", tagFilePath, err)
		}
		fmt.Printf("📕 %s\n", tagFilePath)
	}
}

func GenerateRSSFeed(config Config, posts []PostMetadata) error {
	items := make([]RSSItem, len(posts))
	for i, post := range posts {
		items[i] = RSSItem{
			Title:       post.Title,
			Link:        config.URL + "/" + post.Link + "/",
			Description: post.Description,
			Author:      post.Author,
			PubDate:     post.Published, // .Format(time.RFC1123), // Format for RSS
		}
	}

	rss := RSSFeed{
		Version: "2.0",
		Channel: RSSChannel{
			Title:       config.BlogName,
			Link:        config.URL + "/",
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

	fmt.Printf("📔 %s\n", outputPath)
	return nil
}

func main() {
	var cssFiles, jsFiles []string

	inputDir := flag.String("input", "", "Directory containing Markdown files")
	templatesDir := flag.String("templates", "templates", "Directory containing HTML templates")
	outputDir := flag.String("output", "", "Directory to write generated HTML files")
	indexTemplatePath := flag.String("index", "", "Path to the index HTML template")
	tagsIndexTemplatePath := flag.String("tags-index", "", "Path to the tags index HTML template")
	tagPageTemplatePath := flag.String("tag-page", "", "Path to the tag page HTML template")
	author := flag.String("author", "Anonymous", "Author name for generated pages")
	blogName := flag.String("blog-name", "Example Blog", "Name of Blog")
	copyright := flag.String("copyright", "Copyright Notice Goes Here", "Copyright notice for generated pages")
	language := flag.String("language", "en-us", "Language in this format: en-us")
	flag.Var((*arrayFlag)(&cssFiles), "css", "Specify CSS files to include (repeatable)")
	flag.Var((*arrayFlag)(&jsFiles), "js", "Specify JS files to include (repeatable)")
	url := flag.String("url", "https://example.com", "Root URL without forward slash")
	flag.Parse()

	if *inputDir == "" || *outputDir == "" || *indexTemplatePath == "" || *tagsIndexTemplatePath == "" || *tagPageTemplatePath == "" {
		log.Fatalf("Please specify all required arguments")
	}

	config := Config{
		InputDir:              *inputDir,
		TemplatesDir:          *templatesDir,
		OutputDir:             *outputDir,
		IndexTemplatePath:     *indexTemplatePath,
		TagsIndexTemplatePath: *tagsIndexTemplatePath,
		TagPageTemplatePath:   *tagPageTemplatePath,
		Author:                *author,
		BlogName:              *blogName,
		Copyright:             *copyright,
		Language:              *language,
		CSSFiles:              cssFiles,
		JSFiles:               jsFiles,
		URL:                   *url,
	}

	fmt.Printf("📗 Draft version %s\n", Version)
	processMarkdownFiles(config)
}

type arrayFlag []string

func (i *arrayFlag) String() string {
	return strings.Join(*i, ",")
}

func (i *arrayFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}
