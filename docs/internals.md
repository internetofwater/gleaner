# Gleaner Internals

This document describes the internal workings of Gleaner. The rest of the old documentation has been moved into the [`archive`](./archive/) folder given the fact that some was outdated.

## Walkthrough of the e2e path

1. Read in CLI arguments with Viper
2. Construct `org/` nquad files 
    1. Get all the sources in the gleaner config
    2. Template them into JSON-LD
    3. Convert the JSON-LD into nquads
    4. Put the nq object into minio in the orgs/ subdirectory
3. Load Sitegraphs if they exist
    1. Get all the sources in the gleaner config
    2. For every site in the sitegraph list
4. Summon the sitemaps
    1. Get all the sources in the gleaner config of type API
    2. Run ResRetrieve to pull down the data graphs at resources
        - For each url associated with a given domain, spawn a go routine with `getDomain`

TODO: There is more code after this that can be further documented, but these may be refactored as the codebase matures and is in goroutines making in slightly harder to reason about. 

## Properties

Every time gleaner is ran, the amount of items in the s3 bucket is the same or greater
- There is nowhere in the gleaner codebase that removes objects in the minio store
- Gleaner runs are additive. If you run with one config and then run again with the same config, 

Confusingly, even though there is a `lastmod` tag in the sitemap XML, it appears that gleaner ignores this. If the site is already in the bucket then it is not recrawled, regardless of if the lastmod date is different. 

If the source's jsonld is incorrect, gleaner fall back automatically to trying to use headless chrome to render the page. If that fails then it is logged and nothing is added to the bucket. 

## Gleaner Config

The gleaner config 
