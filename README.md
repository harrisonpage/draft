# Draft

A bare-bones blog generator, nothing more. Inspired by [ʕ•ᴥ•ʔ Bear](https://github.com/HermanMartinus/bearblog/).

Entirely configured with command-line arguments. Given a series of templates, convert Markdown to HTML files.

No support for packaging external files, images, deployment or hosting. Just a site generator.

## Demo

* [harrison.blog](https://harrison.blog)

## Features

* Posts are authored in Markdown
* Program `draft` convers them to HTML
* Build arbitrary sub-pages
* Sample templates provided
* Automatically generates tag pages
* RSS 2.0 feed

## Folders

The following directories are necessary. Directory names can be customized in the configuration file.

* **`badges`**: SVG icons used across the application
* **`templates`**: HTML templates for rendering content
* **`posts`**: Collection of blog posts to be generated. Files are processed in the order of their file names.

To ensure proper ordering, it is recommended to name your blog post files using the following format:

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

## References

* [github.com/myles/awesome-static-generators](https://github.com/myles/awesome-static-generators)

## Author

* [harrison.page](https://harrison.page)

1-dec-2024
