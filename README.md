# pat-api

[![Update Forms Version](https://github.com/la5nta/pat-api/actions/workflows/go.yml/badge.svg)](https://github.com/la5nta/pat-api/actions/workflows/go.yml)

APIs supporting [Pat](https://github.com/la5nta/pat). The [`ghpages` branch](https://github.com/la5nta/pat-api/tree/ghpages) contains files 
which are updated by the scripts in the `main` branch. Initially, this includes a web scraper which determines the latest version of the Winlink
Form Templates.

## Routes

```
- GET https://api.getpat.io/v1/forms/standard-templates/latest
```
