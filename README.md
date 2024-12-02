# Draft

A bare-bones blog generator, nothing more. Inspired by [ʕ•ᴥ•ʔ Bear](https://github.com/HermanMartinus/bearblog/).

Entirely configured with command-line arguments. Given a series of templates, convert Markdown to HTML files.

No support for packaging external files, images, deployment or hosting. Just a site generator.

## Demo

* [harrison.blog](https://harrison.blog)

## Features

* Posts are authored in Markdown
* Program `draft` convers them to HTML
* Example templates are provided
* Automatically generates tag pages
* RSS 2.0 feed

## Environment Variables

Run `build.sh`. (Sets `DRAFT_BUILD_VERSION` and runs `make`.)

## Configuration

Descriptions of each field in `config.yaml`:

- **`input_dir`**: Directory containing Markdown source files

- **`templates_dir`**: Directory containing HTML template files

- **`output_dir`**: Directory where generated HTML files will be written

- **`index_template_path`**: Path to HTML template used for the main index page

- **`tags_index_template_path`**: Path to HTML template used for the tags index page

- **`tag_page_template_path`**: Path to HTML template used for individual tag pages

- **`author`**: Author name, displayed in generated pages

- **`blog_name`**: Blog name, displayed in metadata and header

- **`copyright`**: Copyright notice, displayed in metadata and footer

- **`language`**: Language identifier for the generated pages in the format `language-region` (e.g., `en-us` for English, United States)

- **`css_files`**: List of URLs or file paths for CSS stylesheets to include

- **`js_files`**: List of URLs or file paths for JavaScript files to include

- **`url`**: Root URL of the website, used for generating absolute links

## Usage

```
./draft config.yaml
```

## References

* [github.com/myles/awesome-static-generators](https://github.com/myles/awesome-static-generators)


## Author

* [harrison.page](https://harrison.page)

1-dec-2024
