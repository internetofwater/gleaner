# Gleaner Internals

This document describes the internal workings of Gleaner. It is written after the refactor and e2e testing work. The rest of the old documentation has been deleted if it was outdated or has been moved into the [`archive`](./archive/) folder if it is still useful in some way. 

There is a significant amount of behavior in Gleaner that is still in development or is unused. This document does not cover parts like these that are not needed in Geoconnex. 

## Testing 

All behavior in this repo is covered by tests that run in CI

- unit tests: `go test ./...`
- e2e tests: `go test -tags 'e2e' ./...` (runs [root_test.go](../cmd/root_test.go))

## Walkthrough of the e2e path

1. Read in CLI arguments with Viper
2. Construct `org/` nquad files 
    1. Get all the sources in the gleaner config
    2. Template them into JSON-LD
    3. Convert the JSON-LD into nquads
    4. Put the nq object into minio in the orgs/ subdirectory
3. Load Sitegraphs if they exist (not relevant to Geoconnex)
    1. Get all the sources in the gleaner config
    2. For every site in the sitegraph list
4. Summon the sitemaps
    1. Get all the sources in the gleaner config of type API
    2. Run ResRetrieve to pull down the data graphs (aka jsonld) at a given url
        - For each url associated with a given domain, spawn a go routine with `getDomain`
    3. Sites will be placed in the s3 bucket
5. Run miller to create the graphs
    NOTE: This is now done by Nabu. Believe that all this could is not relevant to us anymore


TODO: There is more code after this that can be further documented, but these may be refactored as the codebase matures and is in goroutines making in slightly harder to reason about. 

## Properties

Every time gleaner is ran, the amount of items in the s3 bucket is the same or greater
- There is nowhere in the gleaner codebase that removes objects in the minio store
- Gleaner runs are additive. If you run with one config and then run again with the same config, 

Confusingly, even though there is a `lastmod` tag in the sitemap XML, it appears that gleaner ignores this. If the site is already in the bucket then it is not recrawled, regardless of if the lastmod date is different. 

If the source's jsonld is incorrect, gleaner fall back automatically to trying to use headless chrome to render the page. If that fails then it is logged and nothing is added to the bucket. 

## Gleaner Config

The gleaner config contains some fields that require extra explanation.

The following is a sample config that is used for Geoconnex. There are more potential fields in a gleaner config but they are not needed for Geoconnex. 


```yaml
context:
  ## If the user wants to cache the context, use a caching document loader
  ## and read from the contextmaps block in this config file
  cache: true
  ## If strict is true then no fixups will be done on the contextmap to make
  ## sure it is valid.
  strict: true
contextmaps:
## this is a filepath relative to the root of the project
## that specifies the jsonld prefix
- file: assets/schemaorg-current-http.jsonld
  prefix: https://schema.org/
gleaner:
  ## mill is used to determine if we should run millers
  ## which are essentially ways of syncing the graph and s3
  ## this is now done with Nabu
  mill: false
  ## just used for logging to identify what run this is
  runid: gleanerbucket
  summon: true

## This section is completely ignored if mill is false
millers:
  graph: true

## Config use for connecting to minio. 
minio:
  accessKey: minioadmin
  address: localhost
  bucket: gleanerbucket
  port: 9000
  ## If region is set in some s3 providers, auth can fail
  region: us
  secretKey: minioadmin
  ssl: false

## THe list of sources that gleaner should crawl
sources:

  ## It doesn't appear that active actually does anything
- active: 'true'
  domain: https://geoconnex.us

  ## Appears to be irrelevant
  ## By default, Gleaner will always fall back to headless if jsonld can't be found at the source
  headless: 'false'
  name: CUAHSI_CUAHSI_HIS_czo_merced_ids__0
  pid: https://gleaner.io/genid/geoconnex
  propername: CUAHSI_CUAHSI_HIS_czo_merced_ids__0
  
  ## You can have either a sitemap index which specifies multiple sitemaps or which is an XML file with multiple URLs
  sourcetype: sitemap
  url: https://geoconnex.us/sitemap/CUAHSI/CUAHSI_HIS_czo_merced_ids__0.xml
- active: 'true'
  domain: https://geoconnex.us
  headless: 'false'
  name: CUAHSI_CUAHSI_HIS_Gongga_ids__0
  pid: https://gleaner.io/genid/geoconnex
  propername: CUAHSI_CUAHSI_HIS_Gongga_ids__0
  sourcetype: sitemap
  url: https://geoconnex.us/sitemap/CUAHSI/CUAHSI_HIS_Gongga_ids__0.xml

## Controls how gleaner actually puts the data in the s3 bucket
summoner:
  # Doesn't appear to be implemented. Intended to allow for crawling only after a certain time
  after: ''
  # milliseconds (1000 = 1 second) to delay between calls (will FORCE threads to 1)
  delay: null
  # URL for headless chrome that gleaner can use to connect to
  headless: 127.0.0.1:9222

  ## Supposed to be full || diff:  If diff compare what we have currently in gleaner to sitemap, get only new, delete missing
  ## CUrrently only full is supported
  mode: full

  ## Controls how many concurrent URLs gleaner can crawl at once (and thus how many goroutines are spawned at once). 
  ## If this is not specified or is 0 then gleaner will hang and never run
  threads: 5
```