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
        *       
    2. Run ResRetrieve to pull down the data graphs at resources
    3. For each url associated with a given domain, spawn a go routine with getDomain

## Properties