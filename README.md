![Docker Image Version (latest by date)](https://img.shields.io/docker/v/phitux/dailytxt?label=latest%20docker%20version)
![Docker Pulls](https://img.shields.io/docker/pulls/phitux/dailytxt)
![GitHub](https://img.shields.io/github/license/PhiTux/DailyTxT)
[!["Buy Me A Coffee"](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/PhiTux)

> [!WARNING]  
> - When you are coming from version 1 (most likely 1.0.15), you MUST read the [Migration Instructions](#migration-instructions) below!
> - Note: A tag like `...-testing.1` is a <ins>**non-stable**</ins> version and (obviously) used for <ins>testing</ins>. Please backup your data and help testing the new version which uses this testing-tag. You can set up notifications for this at the bottom of the settings ❤️

# DailyTxT

DailyTxT is an encrypted Diary/Journal WebApp with several [features](#features). It is written in [Svelte](https://svelte.dev/) and [Go](https://golang.org/) and meant to be run by [Docker](https://www.docker.com/) - supports AMD64 and ARM64.


> 🗣️🌍 You want to add new languages? See [TRANSLATION.md](TRANSLATION.md) for instructions.

> ⁉️ You have questions, new feature requests or want to report bugs? Please open a new [issue](https://github.com/PhiTux/DailyTxT/issues).



## Content

- [Features](#features)
- [Usage Tips](#usage-tips)
- [Installation](#installation)
- [Migration Instructions](#migration-instructions)
- [About encryption and data storage](#about-encryption-and-data-storage)
- [Changelog](#changelog)
- [Start developing](#start-developing)

![DailyTxT Sreenshot](./readme/combined_readme.png)

## Features

- **Encryption**: Everything you write is encrypted before it's written to the server's storage. Even the admin can't read your private stuff!
- **File-Upload**: You can upload arbitrary files for each day (500 MB max each). They are stored encrypted on the server as well. Images are automatically recognized and added to the...
- **Image Viewer**: View all images of a day in a gallery view and in full screen.
- **Markdown**: You can write your entries in markdown and see a live preview.
- **Tags**: You can add tags to your entries for better organization.
- **Search**: You can search for any word, tag or filename in your entries.
- **Custom Templates**: You can create and use custom templates for your entries.
- **Read Mode**: A distraction-free mode for reading your entries of each month.
- **Multi-Language**: DailyTxT is currently available in **<ins>English, German and French</ins>**. New languages can be added easily, see [TRANSLATION.md](TRANSLATION.md) for instructions.
- **Export to HTML**: You can export your entries (including uploaded files) to HTML format.
- **Mobile**: Responsive design for easy use on mobile screen. Additionally: allows installation as a PWA (Progressive Web App) to your Homescreen.
- **Multi-User**: You can create multiple User Accounts. Each account uses its own encryption key.
- **Admin Panel**: You can (among other things) manage users and open registration for 5 minutes.
- **Statistics Panel**: Each user can see some statistics about his entries (among other things there is a GitHub-like statistic of your entry-distribution).
- **Update Notification**: You will be notified about new docker images.


## Usage Tips

- You can swipe left/right in the calendar to change months.
- You can swipe left/right in the date-area above the editor to switch between days.
- Use Alt + Left/Right (or Ctrl + Option + Left/Right on Mac) to switch between days.
- Drag'n'drop files into DailyTxT to upload them.
- The editor saves automatically. Green border means everything is saved, orange border means unsaved changes.
- You can change the order of files (and images!) by dragging them (at the left side) in the file list.
- A yellow dot in the calendar means, that there are uploaded files for this day.
- The orange button in the calendar can highlight the current day.

## Installation

### TLDR

Simply use the `docker-compose.yml` file provided in this repository. Adjust it to your needs and run
```bash
docker compose up -d
```


### More details

Daily is intended to be run with [Docker](https://www.docker.com/) and [Docker Compose](https://docs.docker.com/compose/). You can use the provided `docker-compose.yml` file in this repository. Adjust the environment variables to your needs. 

You see it here:

```yaml
services:
  dailytxt:
    # choose the correct image tag
    image: phitux/dailytxt:2.x.x
    container_name: dailytxt
    restart: unless-stopped
    volumes:
      # Change the left path to your needs
      - ./data:/data
    environment:
      # Create a secret token by running: openssl rand -base64 32
      - SECRET_TOKEN=...

      # If you want to have the json-files pretty-printed, set some indent.
      # (Otherwise just remove the line)
      - INDENT=4

      # Allow new user registrations.
      # I strongly recommend to keep this disabled except for the first user.
      # You can later temporarily enable it again in the admin panel.
      - ALLOW_REGISTRATION=true

      # Set the Admin-Password (for the admin-panel).
      - ADMIN_PASSWORD=your_admin_password

      # After how many days shall the login-cookie expire?
      - LOGOUT_AFTER_DAYS=40

      # Set the BASE_PATH if you are running DailyTxT under a subpath (e.g. /dailytxt).
      # - BASE_PATH=/dailytxt
    ports:
      # Change the left port to your needs.
      # You often would only see 8000:80. But this way, port 8000 is publicly accessible (without TLS!).
      - 127.0.0.1:8000:80
```

## Migration Instructions 

When coming from version 1 (most likely 1.0.15), please follow these instructions carefully!

⚡ **<ins>TLDR:</ins>**
1. **<ins>BACKUP YOUR DATA</ins>** (the whole data directory you used for version 1).
2. Use the new `docker-compose.yml` file and adjust the environment variables accordingly. 
3. Use the <ins>same data directory ('volume') as before</ins>. Start the new docker image.
4. You will probably have to <ins>clear browser cache</ins> (only of the dailytxt-site) to be able to load the new version at all. You will also have to <ins>delete and reinstall</ins> the Apps on your mobile devices (if you used that).
5. Migration starts automatically when an "old" user logs in for the first time.

🔎 **<ins>More details:</ins>**  

When coming from version 1, you will run into server-errors, if you're only updating the docker image tag. This is because version 2 is a full rewrite and exposes a different internal port (now 80).

Additionally, more or less ***all* environment variables have changed**. 

Other than that, an **automatic migration** is implemented. When starting a docker-image of version 2, the server will check, if the data directory contains data from version 1. If so, all "old" data is moved into a subdirectory called "old". Whenever a user now logs in, the server checks, it the user is present in the new data. If not, it checks if the user is present in the "old" data. If so, the migration to the new format is automatically started. In this process, all "old" data is decrypted with the old algorithm and directly re-encrypted with the new one! This works pretty fast and should not take longer than a few seconds per user (heavily depending on the amount of uploaded files).

When all "old" users have logged in once (and it is ensured, that all data has been migrated successfully!), you can delete the "old" subdirectory in your data directory. This is either possible directly in your file system or via the Admin Panel in DailyTxT (there is a button to delete the "old" directory).


## About encryption and data storage

🔒 For encryption ChaCha20-Poly1305 is used.

When a user logs in, a key is derived from his password with Argon2id, it is called the *derived key*. The *derived key* is stored in a http-only cookie and send on every API-call. This key is used to decrypt the user's *encryption key* (which is randomly generated when the user is created). The *encryption key* is used to encrypt/decrypt all data of this user (entries and uploaded files) and never leaves the server. 

When a user changes his password, the *encryption key* is decrypted with the old *derived key* and re-encrypted with a new *derived key* (derived from the new password).

There is no E2E-encryption used on client-side, because the search-functionality would not work then. All data would have to be sent to client-side for searching.

There are also backup-keys available which can be used as a password-replacement. When they are created, they store the *derived key* encrypted with a random *backup key*. These *backup keys* are shown to the user only once and are to be stored safely by him. When a user loses his password, he can use this *backup key* to decrypt the *derived key* and from that the *encryption key*.

All data is stored in json-files. No database is used, because the main goal is to guarantee highest portability and longterm availability of the data.

## Changelog

> [!WARNING]
> Always backup your data before updating to a new version!

I use some sort of [semver](https://semver.org/) for versioning. But I do not guarantee any backward-compatibility! 
In Version A.B.C:
- A is a complete new major version
- B provides new features and languages
- C is for bug fixes.

Additionally there are tags like A.B.C-testing.1 (...testing.2 etc.) for **non-stable** versions. These versions are not meant to be used in production, but for testing new features and bug fixes before they are released in a stable version.

The old version 1 is moved to the [v1 branch](https://github.com/PhiTux/DailyTxT/tree/v1).

---

#### (testing) 2.2.0 (2025-11-x)
```
- Added czech translation [1]
- Added norwegian translation [2]
```
[1]: Thanks to [@MrBurgerCZ](https://github.com/MrBurgerCZ)  
[2]: Thanks to [@Erbros](https://github.com/Erbros)

---

### 2.1.1 (2025-11-10)
```
- Bugfix for update notification
```

### 2.1.0 (2025-11-10)
```
- Added french translation [1]
- Bugfix for translated strings in exported data
- Now auto-reloads the page, when changing the language in user settings
```
[1]: Thanks to [@Lancelot-Enguerrand](https://github.com/Lancelot-Enguerrand)

### 2.0.1 (2025-11-08)
```
- Fixed missing translation [1]
- Bugfix for update notification
```
[1]: Thanks to [@kontaxis](https://github.com/kontaxis) for the pull request!

### 2.0.0 (2025-11-08)
```
- First stable release of version 2 after several testing versions.
Changes to testing are:
- Bugfix and better feedback on migration
- Bugfix when changing month with shortcut
- Changed calendar settings
- Added TB as file-/disksize
```

## Start developing

You need [Go](https://golang.org/) (at least version 1.24) and [Node.js](https://nodejs.org/) (at least version 24) installed.

### Backend
- `cd backend`
- `go mod tidy` (or `go mod download`)
- Set the following environment variables (adjust values):
  - `DATA_PATH=/path/to/data`
  - `SECRET_TOKEN=secret`
  - `ALLOWED_HOSTS='http://localhost:5173,http://127.0.0.1:5173,http://someotherhost:5173'`
  - `INDENT=4`
  - `DEVELOPMENT=true`
  - `ALLOW_REGISTRATION=true`
  - `ADMIN_PASSWORD=adminpassword`
  - `LOGOUT_AFTER_DAYS=40`
- `go build && ./backend`

### Frontend
- `cd frontend`
- `npm install`
- Create the file `vite.config.local.js` with following content (if you need special hosts) - otherwise you have to change the `npm run dev` command in `package.json` to `vite dev` (without `--config ...`):
    ```
    import baseConfig from './vite.config.js';
     
    export default {
      ...baseConfig,
      server: {
        ...baseConfig.server,
        host: true,
        allowedHosts: ['somespecialhost'],
      },
    };
    ```
- `npm run dev`
