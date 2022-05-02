# API Development Guidelines

## Overview

The purpose of this guide is to help you create REST based APIs which are:
  * simple;
  * cohesive;
  * resource based; and
  * internally consistent

Good APIs can be easily understood by humans, and can be easily implemented
into other software. Resource based APIs are defined by _endpoints_ (resources)
which comprise the _nouns_ of the API, and _methods_ (such as GET, PUT, POST,
etc.) which define the _verbs_.

## Creating a new API

Any good API requires a lot of thought and consideration. Good APIs stick
around for years and are difficult to change once released. When designing your
feature, think about what resources you will need and what types of actions
will be performed on those resources. 

Key things to think about:
  * What types of resources will we expose?
  * What are the relationships between the resources?
  * How will the naming scheme work?
  * What is the minimum set of resources that we need?
  * How will the API be consumed?

## Consistency

This document describes an idiomatic way of defining APIs, but it's possible
(and probably likely!) that you may be changing a feature which does not
follow these API guidelines.

Unless you are releasing an entirely new version of the API, you should always
follow the same local style and be consistent with an existing API. Following
the same style makes it easier for developers to understand the API as a whole,
as well as intuitively understand how your feature works and how it relates to
the rest of the API.

Be wary of releasing new versions of an API in a different style. This makes
it difficult for existing integrations to switch to using your new version.
It's often not worth the time or effort to update an existing, working
integration, even if the older version of the API has been deprecated. This
can lead to fragmentation where new consumers of the API use the new version,
but a sizeable amount of integrations stick to using the old version.


## Naming things

Naming resource objects is tricky. What you name the resource may (and
probably will) stick around for a long time. You should:
  * Use simple, concise words;
  * Use unique names (don't overload);
  * Pluralize resources (because they may become collections); and
  * Avoid overly generic names

You should avoid using compound words when naming resources (e.g.
`/v1/access-keys`), however, if you do use a compound word, use kebab case by
joining together words with a '-' instead of using a space.

## Endpoint Hierarchies

When you have determined what resources you will use, you will need to also
determine the paths on which they get exposed. The paths for our current
API tend to be very flat. Some examples include:

| Path                | Description                    |
| :------------------ | :----------------------------- |
| /v1/destinations    | Kubernetes resources           |
| /v1/users           | Users                          |
| /v1/users/%s/grants | Grants for a specific identity |
| /v1/users/%s/groups | Groups for a specific identity |
| /v1/grants          | Authorization grants           |
| /v1/groups          | Groups and group membership    |
| /v1/providers       | Identity providers             |

Try to keep your pathnames flat. Good APIs provide a simple list of base
resources which do not have deep, nested hierarchies. As a rule of thumb,
if modifying a child resource also requires modifying the parent resource,
you should probably make your resource higher up in the hierarchy.

## Method Calls

Method calls are the 'verbs' which determine how to interact with resources. Our API uses standard CRUD (Create, Read,
Update, Delete) methods for performing actions on most resources, however we break up _Read_ into both getting a resource
and listing collections of resources.

| Method           | REST Call                    | Description                     | Request Body | Response Body           |
| :--------------  | :--------                    | :----------                     | :----------- | :------------           |
| Get              | GET `/v1/<resource>/<id>`    | Get a specific resource         | None         | Resource                |
| List             | GET `/v1/<resource>`         | List a collection of resources  | None         | Collection of resources |
| Create           | POST `/v1/<resource>`        | Create a new resource           | Resource     | Resource                |
| Update           | PUT `/v1/<resource>/<id>`    | Update an existing resource     | Resource     | Resource                |
| Delete           | DELETE `/v1/<resource>/<id>` | Delete a resource               | None         | None                    |

Every method call should almost always return a JSON payload as part of the response, typically in the form of the resource
being modified, or as an object which contains a collection of resources. Our API currently does not return other types
of responses, however, other types of responses can be accomodated by using the appropriate response MIME type.

### Getting a resource

The ***Get*** method call takes the ID of a particular resource and returns a JSON object associated with that resource. It's
possible to use the parameter list to modify which parts of the resource are returned, however, it's usually best to avoid this
and return the entire resource object.

When retrieving a resource, always use the HTTP ***GET*** method, and avoid using other method calls such as ***POST***. Also
avoid using the request body for passing any arguments. Embedding arguments into the request body makes it confusing for
developers to know how to make the call correctly.

The call should return a proper JSON response body which includes the resource, and it should be set with the MIME type
_application/json_.

### Listing resources

The ***List*** method call returns a JSON object which contains a collection of resources. Always ensure that
the collection only includes the same type of resources (i.e. do not return collections containing multiple different
resource types). Returning multiple types of resources makes it difficult for clients to deserialize objects correctly.

Retrieving the collection should always be done with the HTTP ***GET*** method, and not using ***POST***. Similar to
the ***Get*** method call, do not use the request body for passing any parameters. Embedding parameters into the request
body makes it difficult for developers to know how to make the request properly. Query parameters may be used as part
of the request URL in order to do help with pagination and faceting.

The return value should be a JSON object that has a member which contains the collection of resources. Don't return the list
directly (i.e. outside of an object). Returning the collection as a member of the object allows additional metadata to be
added to the collection such as pagination which would otherwise be difficult to add in the future. The response should be
set with the MIME type _application/json_.

### Creating a resource

To ***Create*** a new resource, use the HTTP ***POST*** method along with a JSON request body which includes each of the
necessary fields. 

The new resource which was created should be returned as a new JSON document as part of the response body with MIME type
_application/json_. Any fields which were optional in the request should be added by the server, and any timestamp fields
such as `created` and `updated` should be updated with the correct time.

When processing the request, think of what should be the authoritative source of truth. Do you trust the client to be able
to fill in a field? If not, ignore any parts of the resource request which should be filled in by the server. Return an
error if any parts of the request body have nonsensical data which can't be validated.

### Updating a resource

Updating resources should be done with the HTTP ***PUT*** method along with the resource ID as an argument, and a JSON
request body which includes the necessary fields for the resource. Similar to creating a resource, ignore any fields
where the server should be the authoritative source of truth. Any omitted _optional_ fields in the request should revert
to the default setting for the resource. If any fields can't be properly validated, return an error for a response.

We do not currently support the HTTP ***PATCH*** method. In order to update a method, the client should first ***Get***
the resource it wishes to change, modify any necessary fields, and then call the ***Update*** method. If any required fields
are missing, return an error.

Similiar to the ***Create*** method, return the same resource object which has been updated. This should contain
all of the updated fields which are set by the server. The resource should be a JSON document, and have the MIME type
_application/json_.


### Deleting a resource

To ***Delete*** a resource, use the HTTP ***DELETE*** method along with the resource ID to be deleted. If you need to
delete a large number of resources, consider using a _custom method_ using the HTTP ***POST*** method.


### Custom methods

The vast majority of methods for your API should use the standard CRUD methods if possible. This makes it easier for
developers to more easily understand how your API works, and also makes it easier to implement a client which 
works with your API.

There are a few custom methods which tend to crop up from time to time, including:
  * Methods that are part of other protocols (e.g. Oauth2)
  * Long running, complex tasks
  * Bulk operations
  * Search queries

We don't currently have any long running (batch), bulk operations, or search methods as part of our API, but may add them
in the future. This document will be updated with additional requirements and guidelines.

Custom methods should use the HTTP ***POST*** method call. 

## Return Payloads

### Field Naming

Naming fields, similar to naming resources, can be tricky. Use standard naming fields whenever possible. Some
examples include:

| Name           | Format             | Description                      |
| :---           | :-----             | :----------                      |
| created        | RFC 3339 timestamp | Time a resource was created      |
| expires        | RFC 3339 timestamp | Time a resource expires          |
| id             | string             | UID for a resource               |
| lastSeenAt     | RFC 3339 timestamp | Time a resource was last seen    |
| name           | string             | Name of a resource               |
| privilege      | string             | Role or permission               |
| resource       | string             | Resource name                    |
| subject        | string             | UID for a user or group          |
| updated        | RFC 3339 timestamp | Time a resource was last updated |

If you are creating a compound field name, use lower camel case for the JSON field with the first character
lowercase, and capitalize the first letter of each subsequent word (e.g. thisIsAField). Do not use characters
other that ASCII a-z and A-Z inside of the field name.

### Sequential IDs

When returning a resource, avoid using a sequential ID in the ID field. Using sequential IDs allows
callers to potentially guess a particular resource which can lead to leaking information. We
use ["Snowflake IDs"](https://en.wikipedia.org/wiki/Snowflake_ID) in our `infrahq/infra/api` package which you should use instead.

### Payload Structure

When you are returning a payload, keep the following things in mind:
  * Keep things simple
  * Avoid unnecessary deep nested structures inside of resource payloads
  * If you nest one resource into another resource, make certain that it maintains the same structure as if the resource
    were on its own

## Timestamps

For timestamp fields, we follow [RFC 3339](https://datatracker.ietf.org/doc/html/rfc3339) which overlaps
with ISO 8601. Timestamps are always stored in in UTC form. The client is responsible for converting any
timestamp from UTC to local time.

Timestamp fields follow the form:

`YYYY-MM-DDThh:mm:ss.SSSZ`


Date Format:
  * `[YYYY]` represents the four digit year
  * `[MM]` represents to two digit month (pad with zeros)
  * `[DD]` is the two digit day of the month (pad with zeros)

Time Format:
  * `T` represents the beginning of the time part of the timestamp (as opposed to the date)
  * `[hh]` is the hour of the day (00 through 23)
  * `[mm]` is the minute of the hour (00 through 59)
  * `[ss]` represents the number of seconds (00 through 59)
  * `[SSS]` represents the number of milliseconds (000 through 999)
  * `Z` denotes that the timestamp is in UTC (zulu time)

## Errors and Status Codes

Always use the appropriate status code and write good error and status messages when
returning a response. Writing good error messages helps the user determine when something
went wrong, and it helps clients determine how to recover properly from an error.

Error messages should never include details about how the internals of the server work;
these tend to be non-sensical and don't help the client recover successfuly from an
issue.

We currently use the following status codes when the API returns:

| Status | Type                  |
| :----- | :---                  |
| 200    | Success               | 
| 201    | Created               |
| 204    | No content            |
| 400    | Bad request           |
| 401    | Unauthorized          |
| 403    | Forbidden             |
| 404    | Not found             |
| 409    | Duplicate             |
| 500    | Internal server error |
| 502    | Bad gateway           |

We use a standard format for returning an error:
 
```
{
 "code": <Status code>,
 "message": "<Error message>",
 "fieldErrors": [
  {
   "fieldName": [ "<error1>"...<"errorN">]
  }
 ]
}
```

## Documenting the API
  * Make sure you document your API.
  * Ask for help if you need it
  * Open API docs
    * Swagger (openapi.json)

### Modifying Endpoints

An API is a form of contract between you and the users and clients that are using your API. If you change the
API, you are effectively breaking the contract, and can cause clients to no longer work. It's important that you
think about how your API will evolve over time, and to plan for future changes which won't break existing clients.

There are different types of changes which can have different impacts on the API.

#### Generally OK
  * Adding a new method to an endpoint
  * Adding a new resource or collection type

#### Sometimes OK
  * Adding a new field to a resource

#### Almost always bad
  * Removing methods
  * Removing a field from a resource
  * Modifying the behaviour of a method
  * Modifying the order of array elements


## Versioning

If you do have to change your API, there are several things you can do to make it less painful for
developers and clients:
 * Use a <version> component in the path e.g. /v1-beta/users
 * Use "alpha" and "beta" before releasing new APIs
 * Try not to mix and match versions. This makes it difficult for consumers to know which version of the API they're supposed to use
 * The API version is not tied to the semantic version of the product. Don't try to make them the same

