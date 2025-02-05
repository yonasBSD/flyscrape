<br />

<p align="center">

<picture>
  <source media="(prefers-color-scheme: dark)" srcset=".github/assets/logo-alt.png">
  <source media="(prefers-color-scheme: light)" srcset=".github/assets/logo.png">
  <img width="200" src=".github/assets/logo.png">
</picture>

</p>

<br />

<p align="center">
Flyscrape is a command-line web scraping tool designed for those without <br />advanced programming skills, enabling precise extraction of website data. 
</p>

<br />

<p align="center">
<a href="#installation">Installation</a> · <a href="https://flyscrape.com/docs/getting-started">Documentation</a> · <a href="https://github.com/philippta/flyscrape/releases">Releases</a>
</p>


## Demo


<a href="https://www.youtube.com/watch?v=eGk8qFZ9oM4">
  <img src=".github/assets/flyscrape-demo.jpg" style="border-radius: 6px">
</a>

## Features

- **Standalone:** Flyscrape comes as a single binary executable.
- **jQuery-like:** Extract data from HTML pages with a familiar API.
- **Scriptable:** Use JavaScript to write your data extraction logic.
- **System Cookies:** Give Flyscrape access to your browsers cookie store.
- **Browser Mode:** Render JavaScript heavy pages using a headless Browser.
- **Nested Scraping:** Extract data from linked pages within a single scrape.

## Overview

- [Example](#example)
- [Installation](#installation)
    - [Recommended](#recommended)
    - [Homebrew](#homebrew)
    - [Pre-compiled binary](#pre-compiled-binary)
    - [Compile from source](#compile-from-source)
- [Usage](#usage)
- [Configuration](#configuration)
- [Query API](#query-api)
- [Flyscrape API](#flyscrape-api)
    - [Document Parsing](#document-parsing)
    - [File Downloads](#file-downloads)
- [Issues and suggestions](#issues-and-suggestions)

## Example

This example scrapes the first few pages form Hacker News, specifically the New, Show and Ask sections.

```javascript
export const config = {
    urls: [
        "https://news.ycombinator.com/new",
        "https://news.ycombinator.com/show",
        "https://news.ycombinator.com/ask",
    ],

    // Cache request for later.
    cache: "file",

    // Enable JavaScript rendering.
    browser: true,
    headless: false,

    // Follow pagination 5 times.
    depth: 5,
    follow: ["a.morelink[href]"],
}

export default function ({ doc, absoluteURL }) {
    const title = doc.find("title");
    const posts = doc.find(".athing");

    return {
        title: title.text(),
        posts: posts.map((post) => {
            const link = post.find(".titleline > a");

            return {
                title: link.text(),
                url: link.attr("href"),
            };
        }),
    }
}
```

```bash
$ flyscrape run hackernews.js
[
  {
    "url": "https://news.ycombinator.com/new",
    "data": {
      "title": "New Links | Hacker News",
      "posts": [
        {
          "title": "Show HN: flyscrape - An standalone and scriptable web scraper",
          "url": "https://flyscrape.com/"
        },
        ...
      ]
    }
  }
]
```

Check out the [examples folder](examples) for more detailed examples.

## Installation

### Recommended

The easiest way to install `flyscrape` is via its install script.

```bash
curl -fsSL https://flyscrape.com/install | bash
```

### Homebrew

For macOS users `flyscrape` is also available via homebrew:

```bash
brew install flyscrape
```

### Pre-compiled binary

`flyscrape` is available for MacOS, Linux and Windows as a downloadable binary from the [releases page](https://github.com/philippta/flyscrape/releases).

### Compile from source

To compile flyscrape from source, follow these steps:

1. Install Go: Make sure you have Go installed on your system. If not, you can download it from [https://go.dev/](https://go.dev/).

2. Install flyscrape: Open a terminal and run the following command:

   ```bash
   go install github.com/philippta/flyscrape/cmd/flyscrape@latest
   ```

## Usage

```
Usage:

    flyscrape run SCRIPT [config flags]

Examples:

    # Run the script.
    $ flyscrape run example.js

    # Set the URL as argument.
    $ flyscrape run example.js --url "http://other.com"

    # Enable proxy support.
    $ flyscrape run example.js --proxies "http://someproxy:8043"

    # Follow paginated links.
    $ flyscrape run example.js --depth 5 --follow ".next-button > a"

    # Set the output format to ndjson.
    $ flyscrape run example.js --output.format ndjson

    # Write the output to a file.
    $ flyscrape run example.js --output.file results.json
```

## Configuration

Below is an example scraping script that showcases the capabilities of flyscrape. For a full documentation of all configuration options, visit the [documentation page](https://flyscrape.com/docs/getting-started/).

```javascript
export const config = {
    // Specify the URL to start scraping from.
    url: "https://example.com/",

    // Specify the multiple URLs to start scraping from.   (default = [])
    urls: [                          
        "https://anothersite.com/",
        "https://yetanother.com/",
    ],

    // Enable rendering with headless browser.             (default = false)
    browser: true,

    // Specify if browser should be headless or not.       (default = true)
    headless: false,

    // Specify how deep links should be followed.          (default = 0, no follow)
    depth: 5,                        

    // Specify the css selectors to follow.                (default = ["a[href]"])
    // Setting follow to [] disables automatic following.
    // Can later be used with manual following.
    follow: [".next > a", ".related a"],                      
 
    // Specify the allowed domains. ['*'] for all.         (default = domain from url)
    allowedDomains: ["example.com", "anothersite.com"],              
 
    // Specify the blocked domains.                        (default = none)
    blockedDomains: ["somesite.com"],              

    // Specify the allowed URLs as regex.                  (default = all allowed)
    allowedURLs: ["/posts", "/articles/\d+"],                 
 
    // Specify the blocked URLs as regex.                  (default = none)
    blockedURLs: ["/admin"],                 
   
    // Specify the rate in requests per minute.            (default = no rate limit)
    rate: 60,                       

    // Specify the number of concurrent requests.          (default = no limit)
    concurrency: 1,                       

    // Specify a single HTTP(S) proxy URL.                 (default = no proxy)
    // Note: Not compatible with browser mode.
    proxy: "http://someproxy.com:8043",

    // Specify multiple HTTP(S) proxy URLs.                (default = no proxy)
    // Note: Not compatible with browser mode.
    proxies: [
      "http://someproxy.com:8043",
      "http://someotherproxy.com:8043",
    ],                     

    // Enable file-based request caching.                  (default = no cache)
    cache: "file",                   

    // Specify the HTTP request header.                    (default = none)
    headers: {                       
        "Authorization": "Bearer ...",
        "User-Agent": "Mozilla ...",
    },

    // Use the cookie store of your local browser.         (default = off)
    // Options: "chrome" | "edge" | "firefox"
    cookies: "chrome",

    // Specify the output options.
    output: {
        // Specify the output file.                        (default = stdout)
        file: "results.json",
        
        // Specify the output format.                      (default = json)
        // Options: "json" | "ndjson"
        format: "json",
    },
};

export default function ({ doc, url, absoluteURL, scrape, follow }) {
    // doc
    // Contains the parsed HTML document.

    // url
    // Contains the scraped URL.

    // absoluteURL("/foo")
    // Transforms a relative URL into absolute URL.

    // scrape(url, function({ doc, url, absoluteURL, scrape }) {
    //     return { ... };
    // })
    // Scrapes a linked page and returns the scrape result.

    // follow("/foo")
    // Follows a link manually.
    // Disable automatic following with `follow: []` for best results.
}
```

## Query API

```javascript
// <div class="element" foo="bar">Hey</div>
const el = doc.find(".element")
el.text()                                 // "Hey"
el.html()                                 // `<div class="element">Hey</div>`
el.name()                                 // div
el.attr("foo")                            // "bar"
el.hasAttr("foo")                         // true
el.hasClass("element")                    // true

// <ul>
//   <li class="a">Item 1</li>
//   <li>Item 2</li>
//   <li>Item 3</li>
// </ul>
const list = doc.find("ul")
list.children()                           // [<li class="a">Item 1</li>, <li>Item 2</li>, <li>Item 3</li>]

const items = list.find("li")
items.length()                            // 3
items.first()                             // <li>Item 1</li>
items.last()                              // <li>Item 3</li>
items.get(1)                              // <li>Item 2</li>
items.get(1).prev()                       // <li>Item 1</li>
items.get(1).next()                       // <li>Item 3</li>
items.get(1).parent()                     // <ul>...</ul>
items.get(1).siblings()                   // [<li class="a">Item 1</li>, <li>Item 2</li>, <li>Item 3</li>]
items.map(item => item.text())            // ["Item 1", "Item 2", "Item 3"]
items.filter(item => item.hasClass("a"))  // [<li class="a">Item 1</li>]

// <div>
//   <h2 id="aleph">Aleph</h2>
//   <p>Aleph</p>
//   <h2 id="beta">Beta</h2>
//   <p>Beta</p>
//   <h2 id="gamma">Gamma</h2>
//   <p>Gamma</p>
// </div>
const header = doc.find("div h2")

header.get(1).prev()                     // <p>Aleph</p>
header.get(1).prevAll()                  // [<p>Aleph</p>, <h2 id="aleph">Aleph</h2>]
header.get(1).prevUntil('div,h1,h2,h3')  // <h2 id="aleph">Aleph</h2>
header.get(1).next()                     // <p>Beta</p>
header.get(1).nextAll()                  // [<p>Beta</p>, <h2 id="gamma">Gamma</h2>, <p>Gamma</p>]
header.get(1).nextUntil('div,h1,h2,h3')  // <p>Beta</p>
```

## Flyscrape API

### Document Parsing

```javascript
import { parse } from "flyscrape";

const doc = parse(`<div class="foo">bar</div>`);
const text = doc.find(".foo").text();
```

### File Downloads

```javascript
import { download } from "flyscrape/http";

download("http://example.com/image.jpg")              // downloads as "image.jpg"
download("http://example.com/image.jpg", "other.jpg") // downloads as "other.jpg"
download("http://example.com/image.jpg", "dir/")      // downloads as "dir/image.jpg"

// If the server offers a filename via the Content-Disposition header and no
// destination filename is provided, Flyscrape will honor the suggested filename.
// E.g. `Content-Disposition: attachment; filename="archive.zip"`
download("http://example.com/generate_archive.php", "dir/") // downloads as "dir/archive.zip"
```

## Issues and Suggestions

If you encounter any issues or have suggestions for improvement, please [submit an issue](https://github.com/philippta/flyscrape/issues).
