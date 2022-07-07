# Making API Requests

## Version Control

The Infra API is versioned. Requests to the API must contain a header named "Infra-Version".
The best practice is to set this to the version matching the API docs reference you're using, or the version of the server you're using.
Once you set this value you can forget about it until you want to use features from newer API versions.
A valid version header looks like this:

    Infra-Version: 0.13.0

