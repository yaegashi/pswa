---
title: "pswa Introduction"
date: 2021-11-13
---

## Introduction

`pswa` is a small web content/proxy server which is suitable for static web apps.

## Features

- User authentication with Azure Active Directory or many other OpenID Connect IdP
- Flexible authorization using roles based on the `groups` claim in each user's ID token
- Support rewriting, redirecting, and proxying on incoming requests
- Support navigation fallback rewriting suitable for single page apps
- pswa.config.json - JSON configuration file that mimics [staticwebapp.config.json](https://docs.microsoft.com/en-us/azure/static-web-apps/configuration)

```json
{
  "routes": [
    {
      "route": "/admin/*",
      "allowedRoles": ["admin"]
    },
    {
      "route": "/internal/*",
      "allowedRoles": ["authenticated"]
    },
    {
      "route": "/api/*",
      "proxy": "http://localhost:8080/api"
    },
    {
      "route": "/pswa.config.json",
      "redirect": "/"
    }
  ],
  "roles": [
    {
      "role": "admin",
      "members": [
        "34a36796-6043-4dea-85e1-c6ad121a54d4",
        "06fe36df-51ab-49d9-aa3e-2b0034c2cbd1",
        "5bafeeac-804c-4ea4-95c6-11696535c8cb"
      ]
    }
  ]
}
```