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

* **`lang`**: Language as specified in the `<html>` tag e.g. "en"

* **`language`**: Language identifier for the generated pages in the format `language-region` (e.g., `en-us` for English, United States)

* **`css_files`**: List of URLs or file paths for CSS stylesheets to include

* **`js_files`**: List of URLs or file paths for JavaScript files to include

* **`url`**: Root URL of the website, used for generating absolute links

* **`base_path`**: Optional prefix for all URLs, just `blog` for a URL like <https://www.example.com/blog/>

* **`back_label`**: Label for back links

* **`pages`**: List of sub-pages to build

The `pages` block should be in this format:

```
pages:
  - template: about.html
    title: About
    path: about
```

* **`badges`**: List of badges and links

The `badges` block might look like this:

```
badges:
  - title: "Home Page"
    url: "https://harrison.page"
    icon: "home"
  - title: "Photography"
    url: "https://harrison.photography"
    icon: "camera"
```

The `icon` field refers to an SVG file in the `badges` folder. For example, `camera` maps to `badges/camera.svg`.
[Lucide](https://lucide.dev) is an excellent resource for SVG icons.

## Usage

```
./draft config.yaml
```

## SVG Icons

* Courtesy of [Lucide](https://lucide.dev/license)

## References

* [github.com/myles/awesome-static-generators](https://github.com/myles/awesome-static-generators)

## Author

* [harrison.page](https://harrison.page)

1-dec-2024
