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

## Usage

Fill in the blanks:

```
./draft --author "Your Name Here" \
    --blog-name "Generic Blog Name" \
    --copyright "2024" \
    --css "https://example.com/stylesheet.css" \
    --js "https://example.com/script.js" \
    --language "en-us" \
    --output "/var/www/html" \
    --templates templates
    --url "https://example.com"
    --index templates/index.template \
    --tags-index templates/tags-index.template \
    --tag-page templates/tag-page.template
```

## References

* [github.com/myles/awesome-static-generators](https://github.com/myles/awesome-static-generators)


## Author

* [harrison.page](https://harrison.page)

1-dec-2024
