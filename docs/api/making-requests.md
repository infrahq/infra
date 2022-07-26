# Making API Requests

## Version Control

The Infra API is versioned. Requests to the API must contain a header named "Infra-Version".
The best practice is to set this to the version matching the API docs reference you're using, or the version of the server you're using.
Once you set this value you can forget about it until you want to use features from newer API versions.
A valid version header looks like this:

    Infra-Version: 0.13.0

## Pagination

Every List Response in the Infra API is paginated (split into pages). If the page number and limit (page size) aren't specified, then the response will contain the first page of 100 records.

To get the full list of responses, you will need to make multiple requests, specifying the page and limit in the query parameters like so:

* `GET /api/grants?page=2` returns the second page of 100 grants.
* `GET /api/users?page=1&limit=10` returns the first page of 10 users
* `GET /api/users?page=2&limit=10` returns the second page of 10 users


You can use the `totalPages` field to determine the number of pages you will need to request to get all records with the given limit. The maximum limit/page size is 1000.
