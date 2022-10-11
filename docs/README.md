# Infra Documentation

## Overview

Infra's documentation is built with [Markdoc](https://markdoc.io/). All documentation lives in this directory. While it can be read and rendered in GitHub, documentation can include special tags for an enhanced reading experience on [Infra's website](https://infrahq.com/docs) - think of docs as data.

## Building the documentation

To preview the documentation, run the website locally:

```
cd ../website

npm install
npm run dev
```

Then visit `http://localhost:3000/docs`

## Front matter

- `position`: the position in the list
- `title:` the title to display

Example article with front matter:

```md
---
title: My article
position: 4
---

This is an example article with front matter.
```

## Tags

Category directories have a `.category` file in the directory.

## Category front matter

Categories can have the same [front matter](#front-matter) as pages, with the following additional fields in their `.category` file:

- `links` a list of links to other pages

For example:

```yaml
# example .category file
title: My Category
position: 2
links:
  - title: Google
    href: https://google.com
    position: 4
  - title: About
    href: /about
    position: 4
```

### Category pages

Categories can be pages, too, by creating a `README.md` file in the category's directory.

## Components

### Callouts

Callouts are in-article tooltips:

```md
{% callout type="info" %}
For your information
{% /callout %}

{% callout type="warning" %}
This is your final warning
{% /callout %}

{% callout type="success" %}
Congratulations
{% /callout %}
```

### Tabs

Infra's documentation can render content in tabs:

```md
{% tabs %}

{% tab label="macOS" %}
Mac instructions
{% /tab %}

{% tab label="Windows" %}
Windows instructions
{% /tab %}

{% tab label="Linux" %}
Linux instructions
{% /tab %}

{% /tabs %}
```

### YouTube

YouTube videos are embeddable:

```md
{% youtube id="kxlIDUPu-AY /%}
```

### Partials

Pages can include partials. For example, create a `partial.md` file:

```md
## Example partial

This content can be included in different pages.
```

Then use this partial:

```md
{% partial file="partial.md" /%}
```
