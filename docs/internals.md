# Gleaner Internals

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
        - For each url associated with a given domain, spawn a go routine with getDomain

## Properties

Every time gleaner is ran, the amount of items in the s3 bucket is the same or greater
- There is nowhere in the gleaner codebase that removes objects in the minio store

Confusingly, even though there is a ladmod tag in the sitemap XML, it appears that gleaner ignores this. If the site is already in the bucket then it is not recrawled, regardless of if the lastmod date is different. 