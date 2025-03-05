# Gleaner 

This repo is a heavily modified fork of the [gleanerio/gleaner](https://github.com/gleanerio/gleaner) project, used under the Apache 2.0 license. It has been modified to be easier to test for the Geoconnex project.

## About

Gleaner is a tool for extracting JSON-LD from web pages. You provide Gleaner a 
list of sites to index and it will access and retrieve pages based on 
the sitemap.xml of the domain(s). Gleaner can then check for well formed 
and valid structure in documents.  The product of Gleaner runs can then
be used to form Knowledge Graphs, Full-Text Indexes, Semantic Indexes
Spatial Indexes or other products to drive discovery and use.  

_The image below gives an overview of the basic workflow of Gleaner_
 ![A diagram howing a web crawler which harvests data with go, and sends it to s3](./docs/images/gleaner_ad1.png)


This image show that the product of Gleaner is really a populated
data warehouse (document warehouse).  Where those documents are 
either the JSON-LD structured data document harvested or the 
the provenance graphs generated by Gleaner during the process of
harvesting. 

Gleaner talks to an S3 compliant object store as part of its configuration.
This can be AWS S3, Google Cloud Storage (GCS) or other S3 compliant 
object stores.  A typical set up might see the use the open source
Minio package in this role.  

Note also the use of headless chrome in this diagram.  A headless chrome
instance is use for those cases where the resources to be harvested
are placing the JSON-LD documents into the document object model (DOM)
dynamically.   In this case then the headless chrome is used to render 
the page and run the Javascript to form the rendered HTML document that 
can be parsed for the JSON-LD.


## Usage with [Nabu](https://github.com/internetofwater/nabu)

 ![Gleaner usage with nabu that shows both harvesting and syncing operations](./docs/images/gleaner_ad2.png)


