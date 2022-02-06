# DailyTxT

DailyTxT is an encrypted Diary Web-App to simply write down your stories of the day and to easily find them again.
It is written in [Python Flask](https://flask.palletsprojects.com/) (Backend) and [Vue.JS](https://vuejs.org/) (Frontend) and meant to be run via **[Docker](https://hub.docker.com/r/phitux/dailytxt/)**.

You should definitely take a quick look at [How to Use](#how-to-use) to see all the hidden features.

## Chapters

- [Features](#features)
- [How to Use](#how-to-use)
- [Installation](#installation)
- [Changelog](#changelog)

## Features

- **Encryption:** Everything you write is encrypted before it's written to the server's storage. Even the admin can't read your private stuff!
- **File-Upload:** You can upload arbitrary files for each day (128 MB max each). They are **stored encrypted** on the server as well.
- **Search:** Support your memorization by searching all your notes for any keyword to quickly rediscover what you have already forgotten.
- **Multi-Language-Support:** The Web-App is currently available in **German**, **English** and **Spanish** translation. The language gets detected and selected automatically based on your browser's settings. More languages are easy to add - [iff](https://en.wikipedia.org/wiki/If_and_only_if) **you** provide me with the language-files! See `client/src/lang/translations`
- **Mobile:** Responsive design for easy use on mobile screen. Additionally: allows installation "A2HS" (Add 2 Home Screen)
- **Multi-User-Spport:** You can create multiple User Accounts. Each account uses its own encryption key - so there is no risk for a data breach.
- **Backup & Restore:** You can export all data (decrypted) for backup-reasons in a zip-file and reimport the data later by uploading the zip.

## How to Use

There are two cool Shortcuts you should know:

- Move back/forth one day: <kbd>Alt</kbd> + <kbd>&#8592;</kbd> / <kbd>Alt</kbd> + <kbd>&#8594;</kbd>
- Select the search field: <kbd>Ctrl</kbd> + <kbd>F</kbd>

![DailyTxT Description](readme/DailyTxT_Description.jpg)

### Note on backup & restore: 
You can only reimport/restore **all** data at once, that was exported before, it's not selective! The restored data is then the most recent data for each day and any previous data for that day was moved to the day's history.

## Installation

### Quickstart

Simply use the **docker-compose.yml** file from the repo. Adapt it to your needs and run

```bash
docker-compose up -d
```

### More detailed Installation

DailyTxT is inteded to be run via Docker. There is an [official docker image](https://hub.docker.com/r/phitux/dailytxt/).

Use the **docker-compose.yml** file in this repo and adapt it. You see it here:

```yaml
dailytxt:
  image: phitux/dailytxt:latest
  container_name: dailytxt
  restart: always
  environment:
    # That's the internal container-port. You can actually use any portnumber (must match with the one at 'ports')
    - PORT=8765

    - SECRET_KEY=<openssl rand -base64 32>

    # Set it to False or remove the line completely to disallow registration of new users.
    - ALLOW_REGISTRATION=True

    # Use this if you want the json log file to be indented. Makes it easier to compare the files. Otherwise just remove this line! 
    - DATA_INDENT=2

    # Set after how many days the JWT token will expire and you have to re-login. Defaults to 30 days if line is ommited.
    - JWT_EXP_DAYS=60
  ports:
    - "127.0.0.1:<host_port>:8765"
    # perhaps you only want:
    # "<host_port>:8765"
  volumes:
    - "</save/my/dailytxt-files/here/>:/app/data/"
```

There are **four <ins>important</ins> things that you should adapt** to your needs:

- Generate a **secret_key** by running `openssl rand -base64 32` in your bash
- You probably want to allow the **registration** on the first run, generate the necessary accounts, and then restart the container with disallowed registration!
- Choose the **port** on your localhost.
- Select a **path**, where your diary-files are saved beyond the container lifetime.

Then simply run

```bash
docker-compose up -d
```

---

**If you do NOT use docker-compose**, then simply run the following equivalent from bash (and see the info above to adapt it to your needs!):

```bash
docker run -p 127.0.0.1:<host_port>:8765 -e "PORT=8765" -e "SECRET_KEY=<openssl rand -base64 32>" -e "ALLOW_REGISTRATION=True" -v </save/my/dailytxt-files/here/>:/app/data --name dailytxt -d phitux/dailytxt:latest
```

#### ProxyPass

You will probably proxypass the container via Nginx or Apache2.
Here is an example apache-configuration (which of course needs to be slightly adapted):

```apache
<VirtualHost *:443>
    ServerName dailytxt.mydomain.tld

    SSLEngine On
    <MY-CERT-PATHS.........>

    Header always set Strict-Transport-Security "max-age=63072000; includeSubDomains"
    Header append X-FRAME-OPTIONS SAMEORIGIN

    ProxyPreserveHost On
    ProxyRequests off
    AllowEncodedSlashes NoDecode
    ProxyPass / http://localhost:8007/ nocanon
    ProxyPassReverse / http://localhost:8007/
</VirtualHost>
```

You can now (Version >= 1.0.4) also run DailyTxT on a subpath.

---
---

## Changelog

### 1.0.8
(2022_02_06)\
*Actually a **big** update!*
```
- Added spanish translation [1]
- Added possibility to reimport (restore) data, that was exported before! 
- Added possibility to create Backup Codes. They work like one-time passwords.
```
[1]: Thanks to [@SmartPhoneLover](https://github.com/SmartPhoneLover)

---

### 1.0.7
(2022_01_12)
```
- Added image preview for uploaded image-files [1]
- Bugfix: couldn't save text, when there was not yet any text for this day BUT already an uploaded file.
- Added automatic update-check with notification for new docker versions (this is sort of beta, let's see how well it works...)
```
[1]: Thanks to [@rhld16](https://github.com/rhld16)

---

### 1.0.6
(2021_12_12)
```
Docker image size was drastically reduced by new multi-stage build.
```
Thank you very much [@rhld16](https://github.com/rhld16)!

---

### 1.0.5
(2021_12_12)
```
Now automatically redirects (in most cases) to login-page when the Webtoken expired.
```

---

### 1.0.4 
(2021_12_12)
```
Added possibility to run application on a subpath
```
Thanks to [@rhld16](https://github.com/rhld16)'s PR

---

### 1.0.3
```
Added two new environment variables:
1. DATA_INDENT [1]
2. JWT_EXP_DAYS
(see explanation above)
```
[1]: Thanks to [@bogosj]( https://github.com/bogosj)'s PR

---

### 1.0.2
```
Minor CSS Updates.
Exported Zip now also includes the history texts.
```

---

### 1.0.1 
```
Initial Release
```

## License

MIT, see the License-File.
