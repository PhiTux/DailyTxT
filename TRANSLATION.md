# Translate DailyTxT into your language

You want to help to translate DailyTxT into your language? Great! Thank you very much!  
This document provides a description how you can help.

## How to translate

There are **<ins>two ways</ins>** to translate DailyTxT:

1. Provide a translation by yourself and create a <ins>Pull request</ins>.
2. Use <ins>Tolgee</ins> to provide a translation.

### 1. Provide a translation by yourself and create a Pull request

1. Fork the repository.
2. You find the translation files in the folder `frontend/src/i18n/`. All strings of a language are stored in a single file!
3. Grab some AI tool, throw 'en.json' (or any other language file) in and ask to translate it into your language (then double-check it of course). Or do it all by yourself.
4. Save this file as `<your-language-code>.json` (e.g. `de.json` for German, `fr.json` for French, `es.json` for Spanish,...). You should only add the specific country code (like `de-CH` for Swiss-German or `fr-BE` for Belgian French) if there are significant differences in the language (try to make it as general as possible).
5. Open the file `flags.json` in the same folder and add an entry for your language. The value of each entry (the right side) should be the emoji-flag of your country. Some browsers do not support emoji-flags, but you can find the flags on websites like this: https://flagpedia.net/emoji
6. Open the file `frontend/src/routes/+layout.svelte`. Right at the beginning of the page (at around line 25), there are a few occurences of the supported languages. Add your language here, too. Thank you ❤️
7. Create a Pull request.
8. I will then publish a testing image on Docker Hub for you (named like 2.0.0-testing.1) to test your translation (and perhaps other changes too). If everything is fine, it will stay as it is and be part of the next release. Otherwise we can repeat the last steps.

### 2. Use Tolgee to provide a translation

> [!NOTE]  
> I use a free account of [Tolgee](https://tolgee.io). Therefore, there is sadly a limit of 3 concurrent users (translators) per project (me included). So when you finished translating and everything is working fine, I will have to delete you from the project to make space for other translators. Sorry for that.

1. Open an issue in this repository and tell me that you want to help translating DailyTxT into your language. Tell me which language you want to translate (e.g. German, French, Spanish,...). If possible, please provide a language code, that describes you language <ins>as general as possible</ins> (e.g. `de` for German, `fr` for French, `es` for Spanish,...). You should only add the specific country code (like `de-CH` for Swiss-German or `fr-BE` for Belgian French) if there are significant differences in the language.
2. I will then send you an invitation-link to join the Tolgee project.
3. Open the link, create a Tolgee account (if you do not have one yet) and join the project.
4. You can now start translating! You find all strings in the project. You can filter for your language and see which strings are not translated yet.  
<ins>Important:</ins> You have <ins>several</ins> possibilities how to use Tolgee:
   1. You can either translate it directly in the Tolgee interface.
   2. Or you can download the JSON-file of your language, translate it with an AI tool (or by yourself) and upload it again. (Thats pretty similar to the first way of translating, but you do not have to create a Pull request by yourself.)
   3. Install the "Tolgee Tools" Browser Extension (Available for Firefox and Chrome-based browsers) and translate a lot of strings directly in the browser-frontend. You have to create an API-Key for that and use the URL "https://app.tolgee.io" in the Addon. Sadly several strings (e.g. inside a modal) cannot be translated with the browser extension. But you can use the extension for a lot of strings and use the Tolgee interface (Website) for the rest.
5. When you are done, please tell me in the issue you created at the beginning.
6. I will then publish a testing image on Docker Hub for you (named like 2.0.0-testing.1) to test your translation (and perhaps other changes too). If everything is fine, it will stay as it is and be part of the next release. Otherwise we can repeat the last steps.


# Developer notes

Check the following:
- `frontend/src/i18n/flags.json`: Must contain an entry for the new language with the corresponding emoji-flag. See eg.: https://flagpedia.net/emoji
- `frontend/src/routes/+layout.svelte`: Must contain an entry for the new language at around line 25.
