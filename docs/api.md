# API Reference

## Contents

### Overview
- [Authentication](#authentication)
- [Pagination](#authentication)

### Core Resources
- [Users](#users)
- [Tokens](#tokens)
- [Providers](#providers)
- (Coming Soon) [Groups](#roles)
- (Coming Soon) [Groups](#groups)
- (Coming Soon) [Destinations](#destinations)

## Overview

### Authentication

### Pagination

## Core Resources

### Users

```
  POST /v1/users
  POST /v1/users/:id
   GET /v1/users
   GET /v1/users/:id
DELETE /v1/users/:id
```

### Tokens

Tokens are used to access [Destinations](#destinations).

```
  POST /v1/tokens
  POST /v1/tokens/:id
   GET /v1/tokens
   GET /v1/tokens/:id
DELETE /v1/tokens/:id
```

### Destinations

Destinations are infrastructure endpoints that

```
  POST /v1/destinations
  POST /v1/destinations/:id
   GET /v1/destinations
   GET /v1/destinations/:id
DELETE /v1/destinations/:id
```

### Providers

Providers are used to automatically create and authenticate users. Infra comes with a default local provider (that uses username + password).

```
  POST /v1/providers
  POST /v1/providers/:id
   GET /v1/providers
   GET /v1/providers/:id
DELETE /v1/providers/:id
```
