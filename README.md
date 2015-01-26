# APROXY
An Advanced Application Proxy

### Introduction

AProxy (An Advanced Application Proxy) is a way to translate complex and generic REST requests into more simplified and specific requests. 

Features:
- scalable "hot plugging/unplugging" of advanced request proxy/re-write configurations (also called Mappings)
- advanced url and body generation using [GO Templates](http://golang.org/pkg/html/template/)

### Example

The following complex request

```
POST http://api.com/service/application/challenge_result/_search?search_type=count&pretty
{
"aggs": {
    "most_played_challenges": {
      "terms": {
        "field": "_parent",
        "order" : { "_count" : "desc" }
      }
    }
  }
}
```
Can be turned into a simple request
```
GET http://application.com/challenges/popular
```
By using the following configuration (or Mapping)
```json
{
  "mappings" : {
    "indexer" : {
      "target" : {
        "headers" : {
          "Content-Type" : "application/json; charset=UTF-8"
        },
        "verb" : "POST",
        "uri" : "http://api.com/service/application/challenge_result/_search?search_type=count&pretty",
        "body" : "
        {
          \"aggs\"": {
            \"most_played_challenges\": {
              \"terms\": {
                \"field\" : \"_parent\",
                \"order\" : { \"_count\" : \"desc\" }
              }
            }
          }
        }
      "
      },
      "mapping" : {
        "request.path" : "(?i)^/challenges/popular/?$"
      }
    }
  }
}
```

### Mappings

AProxy mappings take the form of
```json
{
  "mappings" : {
    "<NAME>" : {
      "target" : {
        "headers" : {"<HEADER_NAME>" : "<HEADER_VALUE>"},
        "verb" : "GET|POST|PUT|DELETE|OPTION|HEAD",
        "uri" : "<URI_TEMPLATE>",
        "body" : "<BODY_TEMPLATE>"
      },
      "mapping" : {
        "<PROPERTY_NAME1>" : "<REGULAR_EXPRESSION>"
      }
    }
  }
}
```
Each mapping has a __<NAME>__ which is a unique identifier.
Every mapping has a __target__ property:
- __headers__ a list of headers to send to the underlying service
- __verb__ verb to use in request to the underlying service
- __uri__ uri to the underlying service, this is a [template](http://gohugo.io/templates/go-templates/) and not a simple string
- __body__ body to send to the underlying service, this is a [template](http://gohugo.io/templates/go-templates/) and not a simple string

Every mapping also has a __mapping__ property, this is a map of property names and regular expressions used to map this particular mapping to incoming requests.

properties available for mappings and templates are:
- __request.method__	request verb, ex: GET, POST, PUT, OPTION, DELETE, HEAD
- __request.path__	ex: /twitter/123451
- __request.host__	ex: www.google.com
- __request.uri__	raw uri, ex: /twitter/123451?id=4512&ref=sau
- __request.content-length__	ex: 1024
- __query.xxx__	always lower-cased, ex: /twitter/123451?id=4512&ref=sau will avail query.id and query.ref
- __header.xxx__	always lower-cased, ex: header.content-type, header.user-agent
