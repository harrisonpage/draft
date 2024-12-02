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

Run `build.sh` or otherwise set `DRAFT_BUILD_VERSION` and run `make`.

## Configuration

Below are descriptions of each field in `config.yaml`

- **`input_dir`**: Specifies the directory containing Markdown source files to process.

- **`templates_dir`**: Specifies the directory containing HTML template files used for generating output.

- **`output_dir`**: Defines the directory where the generated HTML files will be written.

- **`index_template_path`**: Path to the HTML template used for the main index page.

- **`tags_index_template_path`**: Path to the HTML template used for the tags index page.

- **`tag_page_template_path`**: Path to the HTML template used for individual tag pages.

- **`author`**: Name of the author to display in the generated pages.

- **`blog_name`**: The name of the blog, displayed in metadata or header sections.

- **`copyright`**: Copyright notice to include in the generated pages.

- **`language`**: The language identifier for the generated pages in the format `language-region` (e.g., `en-us` for English, United States).

- **`css_files`**: A list of URLs or file paths for CSS stylesheets to include in the generated pages.

- **`js_files`**: A list of URLs or file paths for JavaScript files to include in the generated pages.

- **`url`**: The root URL of the website. This is used for generating absolute links in the output.

## Usage

```
./draft config.yaml
```

## References

* [github.com/myles/awesome-static-generators](https://github.com/myles/awesome-static-generators)


## Author

* [harrison.page](https://harrison.page)

1-dec-2024
