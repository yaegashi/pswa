# pswa - Protected Static Web App

## Introduction

pswa is a simple web content/proxy server which is suitable for various static web apps.

## Features

- Available as a Docker image `ghcr.io/yaegashi/pswa`
- User authentication with Azure Active Directory (supporting other OIDC providers is planned)
  - Support Azure App Service authentication (aka Easy Auth)
- Flexible authorization using roles based on the following member groups sources:
  - `groups` claim in each user's ID token
  - `/me/getMemberObjects` Microsoft Graph API call
- Support rewriting, redirecting, and proxying on incoming requests
- Support navigation fallback rewriting suitable for single page apps
- pswa.config.json - JSON configuration file that mimics [staticwebapp.config.json](https://docs.microsoft.com/en-us/azure/static-web-apps/configuration)

## Configuration

### Azure AD authentication settings

When running with the built-in Azure AD authentication support:
- Register an Azure AD application with [Azure Portal](https://portal.azure.com).
- Collect tenant ID and client ID for `PSWA_TENANT_URI` and `PSWA_CLIENT_ID` settings respectively.
- Generate a client secret string for `PSWA_CLIENT_SECRET` settings.
- Add `<YOUR-APP-URL>/.auth/pswa/callback` to the application redirect URIs.  It's also for `PSWA_REDIRECT_URI` settings.

When running on Azure App Service with the authentication enabled (aka Easy Auth):
- Set up [the Easy Auth Azure AD provider](https://learn.microsoft.com/en-us/azure/app-service/configure-authentication-provider-aad).  You will have a dedicated Azure AD application.
- Azure AD related settings in the environment variables are not needed.

### Environment variable settings

See [pswa-example.env](pswa-example.env) for example settings.

|Variable|Description|
|---|---|
|PSWA_TENANT_ID|Tenant ID of Azure AD <sup>*</sup>|
|PSWA_CLIENT_ID|Client ID registered in Azure AD  <sup>*</sup>|
|PSWA_CLIENT_SECRET|Client secret generated in Azure AD  <sup>*</sup>|
|PSWA_REDIRECT_URI|Rediect URI specifed in Azure AD <sup>*</sup>|
|PSWA_AUTH_PARAMS|Additional authorize endpoint parameters in the form of `key1=val1&key2=val2&key3=val3` <sup>*</sup>|
|PSWA_SESSION_KEY|Ramdom string to encrypt values in the cookie session store|
|PSWA_LISTEN|Server address to listen.  Default: `:8080`|
|PSWA_WWW_ROOT|Web content root directory.  Default: `/home/site/wwwroot`|
|PSWA_TEST_ROOT|Web content root directory for tests.  Default: `/testroot`|
|PSWA_CONFIG|Configuration file location.  It's relative to `PSWA_WWW_ROOT` if not an absolute path.  Default: `pswa.config.json`|

<sup>*</sup> Azure AD related settings are not necessary when it runs on [Azure App Service with the authentication enabled](https://learn.microsoft.com/en-us/azure/app-service/overview-authentication-authorization).

### Configuration file (pswa.config.json)

See [pswa-example.config.json](pswa-example.config.json) for example settings.
It's similar to [staticwebapp.config.json of Azure Static Web Apps](https://docs.microsoft.com/en-us/azure/static-web-apps/configuration).

- If `testHandler` is true, it enables the test handler for debugging purposes.
- If `testRoot` is true, it serves web content from `/testroot` instead of `/home/site/wwwroot`.
- You should specify `navigationFallback` to serve an SPA.
- `roles` defines the roles and its members.  `members` are object IDs of Azure AD groups.

```json
{
  "testHandler": true,
  "testRoot": true,
  "navigationFallback": {
    "rewrite": "/index.html",
    "exclude": [
      "/index.html",
      "/*.{js,css,map}"
    ]
  },
  "routes": [
    {
      "route": "/admin/*",
      "allowedRoles": [
        "admin"
      ]
    },
    {
      "route": "/authenticated/*",
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

You can use a [devcontainer](.devcontainer) with docker-in-docker privilege to develop the pswa executable and container.
Follow the steps below in your devcontainer.

Copy [pswa-example.env](pswa-example.env) to pswa.env and edit it for your environment settings:
```console
$ cp pswa-example.env pswa.env
$ vi pswa.env
```

Copy [pswa-example.config.json](pswa-example.config.json) to pswa.config.json and edit it for your site config:
```console
$ cp pswa-example.config.json pswa.config.json
$ vi pswa.config.json
```

Build a container and run it with docker compose:
```console
$ docker compose up --build
[+] Building 1.5s (21/21) FINISHED
...
[+] Running 2/2
 ✔ Network pswa_default   Created
 ✔ Container pswa-pswa-1  Created
Attaching to pswa-pswa-1
pswa-pswa-1  | 2023-06-19T05:00:23.750Z INFO Reading config: /home/site/wwwroot/pswa.config.json
pswa-pswa-1  | 2023-06-19T05:00:23.755Z INFO OpenID Connect auth config:
pswa-pswa-1  | 2023-06-19T05:00:23.755Z INFO   TenantID    = 3822b9ab-ab2c-4f20-a8cd-abe6ac986c37
pswa-pswa-1  | 2023-06-19T05:00:23.755Z INFO   ClientID    = 19c3bf12-a48a-4b68-93f4-353631f95924
pswa-pswa-1  | 2023-06-19T05:00:23.755Z INFO   RedirectURI = http://localhost:8080/.auth/pswa/callback
pswa-pswa-1  | 2023-06-19T05:00:23.755Z INFO   AuthParams  = prompt=select_account
pswa-pswa-1  | 2023-06-19T05:00:24.167Z WARN TestRoot enabled
pswa-pswa-1  | 2023-06-19T05:00:24.167Z INFO Serving from root path /testroot
pswa-pswa-1  | 2023-06-19T05:00:24.167Z INFO Serving on :8080
```

Open `http://localhost:8080` with your web browser.
