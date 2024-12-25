# Draft

A bare-bones static site and blog generator, nothing more.

Entirely configured with command-line arguments. Given a series of templates, convert Markdown to HTML files.

No support for packaging external files, images, deployment or hosting. Just a site generator.

## Demo

* [harrison.blog](https://harrison.blog)

## Features

* Posts are authored in Markdown files, no CMS or database required
* Driven by a YAML configuration file
* A single statically-linked binary converts Markdown to HTML files
* Builds simple sub-pages
* Sample templates provided
* Automatically generates tag pages
* Bring your own CSS files or use [drop-in CSS frameworks](https://github.com/swyxio/spark-joy/blob/master/README.md#drop-in-css-frameworks)
* Generates an RSS 2.0 feed
* Minimal dependencies to build, zero dependencies to run
* Allows for drafts: Set a post's status to `private` and it will be skipped
* Home page shows latest post with an index of all posts
* Headers and footers stored in a common template file
* Compliant with W3C standards for XHTML, CSS and RSS validation
* Minimal code base: Under 1,000 lines of Golang
* Fast page generation
* Easily deploy the `output` folder to production
* Collision detection to avoid unintentionally overwriting same-named posts
* SEO features including meta tags (OpenGraph, etc), sitemap and custom URLs

## Folders

The following directories are necessary. Directory names can be customized in the configuration file.

* **`badges`**: SVG icons used across the application
* **`templates`**: HTML templates for rendering content
* **`posts`**: Collection of blog posts to be generated
* **`output`**: HTML files are saved here

Posts are processed and rendered in the order of their file names, new to old. To ensure proper ordering, it is recommended to name your blog post files using the following format:

```text
YYYYMMDD.descriptive.title.md
```

Example:

```text
20250101.happy.new.year.md
```

## Environment Variables

Run `build.sh`. (Sets `DRAFT_BUILD_VERSION` and runs `make`.)

## Configuration

Descriptions of each field in `config.yaml`:

* **`input_dir`**: Directory containing Markdown source files

* **`templates_dir`**: Directory containing HTML template files

* **`output_dir`**: Directory where generated HTML files will be written

* **`index_template_path`**: Path to HTML template used for the main index page

* **`tags_index_template_path`**: Path to HTML template used for the tags index page

* **`tag_page_template_path`**: Path to HTML template used for individual tag pages

* **`author`**: Author name, displayed in generated pages

* **`blog_name`**: Blog name, displayed in metadata and header

* **`copyright`**: Copyright notice, displayed in metadata and footer

* **`description`**: Brief description, one paragraph or so, used in link unfurls

* **`email`**: Email address, optional. Appears in a meta tag in the header and also wrapped with an `<address>` tag in the footer

* **`lang`**: Language as specified in the `<html>` tag e.g. "en"

* **`language`**: Language identifier for the generated pages in the format `language-region` (e.g., `en-us` for English, United States)

* **`css_files`**: List of URLs or file paths for CSS stylesheets to include

* **`js_files`**: List of URLs or file paths for JavaScript files to include

* **`url`**: Root URL of the website, used for generating absolute links

* **`base_path`**: Optional prefix for all URLs, just `blog` for a URL like <https://www.example.com/blog/>

* **`back_label`**: Label for back links

* **`pages`**: List of sub-pages to build

* **`badges`**: List of badges and links

* **`fediverse_creator`**: Optional Mastodon username e.g. `@harrisonpage@defcon.social`

## Pages

The `pages` block should be in this format:

```yaml
pages:
  - template: about.html
    title: About
    link: about
```

The about.html template will be written here: .../about/index.html.

## Badges

The `badges` block might look like this:

```yaml
badges:
  - title: "Photography"
    url: "https://harrison.photography"
    icon: "camera"
```

The `icon` field refers to an SVG file in the `badges` folder. For example, `camera` maps to `badges/camera.svg`.

[Lucide](https://lucide.dev) is an excellent resource for SVG icons.

## Fediverse Integration

* If the `fediverse_creator` configuration field is filled out, a `fediverse:creator` header will appear on all pages

## Templates

The `templates` folder contains these files:

* **`default.html`**: Post template, can be changed per-post
* **`index.html`**: Home page, contains most recent post and an index
* **`tags.html`**: List of all tags used, example [here](https://harrison.blog/tags/)
* **`tag.html`**: Page showcasing individual tags, example [here](https://harrison.blog/tags/code/)
* **`shared.html`**: Top and bottom matter shared among all pages

### Examples

These templates are examples of custom pages as specified in the configuration file.

* about.html
* colophon.html

## Posts

A blog post has a header and a body. The header is surrounded by three dashes: YAML front matter.

Example:

```markdown
---
title: Hello World
link: hello-world
description: Example description
tags: meta
image: https://cdn.harrison.photography/hello/IMG_9587x1920.jpeg
published: 2024-11-29T18:29:00-08:00
template: default.html
favicon: 👋🏻
author: harrison
status: public
---

## Hello World

Welcome to my blog. There are many like it, but this one is mine.
```

Fields:

* **`title`**: Post title, shown in HTML title tag and at the top of the page
* **`link`**: Name of directory the post is served from e.g. http://example.com/hello-world/
* **`description`**: Brief description of your post
* **`tags`**: List of tags separated by comma e.g. `meta,code`
* **`image`**: URL to an image (optional)
* **`published`**: Post date in ISO 8601 format
* **`template`**: Name of a file in the `templates` folder
* **`favicon`**: Emoji associated with post (optional)
* **`author`**: Post author (optional)
* **`status`**: `public` or `private`

## Usage

```text
./draft config.yaml
```

## SVG Icons

* Courtesy of [Lucide](https://lucide.dev/license)

## Prior Art

Many other static site generators out there:

* [awesome-static-generators](https://github.com/myles/awesome-static-generators)

### Noteworthy

I liked the implementation or the approach of these tools:

* [trofaf](https://github.com/mna/trofaf)
* [Eleventy](https://www.11ty.dev)
* [Aurora](https://github.com/capjamesg/aurora)
* [bashblog](https://github.com/cfenollosa/bashblog)
* [Hexo](https://github.com/hexojs/hexo)

## See Also

* [IndieWeb](https://indieweb.org)
* Inspired by [ʕ•ᴥ•ʔ Bear](https://github.com/HermanMartinus/bearblog/)

## Contributing

Contributions are welcome!

* Open an issue or submit a PR
* Contact [me](https://harrison.page) with any questions or suggestions

## History

* Project started on 1-Dec-2024
