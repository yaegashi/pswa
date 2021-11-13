# pswa - Protected Static Web App

## Introduction

pswa is a simple web content/proxy server which is suitable for various static web apps.

## Features

- Available as a Docker image `ghcr.io/yaegashi/pswa`
- User authentication with Azure Active Directory (supporting other OIDC providers is planned)
- Flexible authorization using roles based on the `groups` claim in each user's ID token
- Support rewriting, redirecting, and proxying on incoming requests
- Support navigation fallback rewriting suitable for single page apps
- pswa.config.json - JSON configuration file that mimics [staticwebapp.config.json](https://docs.microsoft.com/en-us/azure/static-web-apps/configuration)

## Configuration

### Azure AD app registration

- Register an Azure AD application with [Azure Portal](https://portal.azure.com).
- Collect tenant ID and client ID for `PSWA_TENANT_URI` and `PSWA_CLIENT_ID` settings respectively.
- Generate a client secret string for `PSWA_CLIENT_SECRET` settings.
- Add `<YOUR-APP-URL>/.auth/login/aad/callback` to the application redirect URIs.  It's also for `PSWA_REDIRECT_URI` settings.
- Modify the application's manifest to set `groupMembershipClaims` to `SecurityGroup`.
It's required to expose user's member groups in ID token.
[See the document for details](https://docs.microsoft.com/en-us/azure/active-directory/develop/reference-app-manifest#groupmembershipclaims-attribute).

### Environment variable settings

|Variable|Description|
|---|---|
|PSWA_TENANT_ID|Tenant ID of Azure AD|
|PSWA_CLIENT_ID|Client ID registered in Azure AD|
|PSWA_CLIENT_SECRET|Client secret generated in Azure AD|
|PSWA_REDIRECT_URI|Rediect URI specifed in Azure AD|
|PSWA_SESSION_KEY|Ramdom string to encrypt values in the cookie session store|
|PSWA_LISTEN|Server address to listen.  Default: `:8080`|
|PSWA_WWWROOT|Web content root directory.  Default: `/home/site/wwwroot`|
|PSWA_CONFIG|Configuration file location.  It's relative to `PSWA_WWWROOT` if not an absolute path.  Default: `pswa.config.json`|

### Configuration file (pswa.config.json)

`roles` configuration is to define roles and its members.
`members` are object IDs of Azure AD groups.

`routes` configuration is very similar to [staticwebapp.config.json of Azure Static Web Apps](https://docs.microsoft.com/en-us/azure/static-web-apps/configuration).

```json
{
  "routes": [
    {
      "route": "/admin/*",
      "allowedRoles": [
        "admin"
      ]
    },
    {
      "route": "/internal/*",
      "allowedRoles": [
        "authenticated"
      ]
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

## Hacking

The following steps need docker, docker-compose, and hugo on your system.

Copy [pswa-example.env](pswa-example.env) to pswa.env and edit it as your environment settings:
```console
$ cp pswa-example.env pswa.env
$ vi pswa.env
```

Edit [pswa.config.json](hugo/static/pswa.config.json) and run hugo to build a static web site in `hugo/public`:
```console
$ vi hugo/static/pswa.config.json
$ hugo -s hugo
```

Build the container and run it with docker-compose:
```console
$ docker-compose up --build
Attaching to pswa_pswa_1
pswa_1  | 2021/11/13 05:15:27 Listening on :8080
```

Open `http://localhost:8080` with your web browser.
