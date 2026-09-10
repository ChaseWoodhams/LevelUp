# English-only language policy

LevelUp has one active application locale: `en`.

- Web and study locale types accept only `en`.
- Manifest files and inline dictionaries contain English values only.
- Formatting uses `en-US`; requests do not negotiate a language.
- Public API DTOs expose English field names and English values only.
- New database writes use English metadata. Historical translation columns and
  migrations may remain when they are required to upgrade existing databases.
- Do not add language selectors, locale branches, paired translation fields, or
  language-specific routes.

When changing user-facing copy, update the English source manifest or the
English-only feature dictionary and regenerate checked-in output as required.
