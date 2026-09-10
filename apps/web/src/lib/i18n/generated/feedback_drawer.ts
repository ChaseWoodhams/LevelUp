// Auto-genere par scripts/build_i18n_manifests.mjs - NE PAS EDITER A LA MAIN.
// Source : apps/web/src/lib/i18n/manifests/feedback_drawer.toml

export const feedbackDrawerManifest = {
  "feedback_drawer.attach.label": { en: "Attach technical info" },
  "feedback_drawer.attach.preview_summary": { en: "Markdown preview" },
  "feedback_drawer.classification.preview": { en: "Type {type} · Severity {severity} · Area {area}" },
  "feedback_drawer.field.description": { en: "Description" },
  "feedback_drawer.field.description_placeholder": { en: "What happened, what you expected, steps to reproduce…" },
  "feedback_drawer.field.title": { en: "Title" },
  "feedback_drawer.field.title_placeholder": { en: "Sum up your feedback in a few words" },
  "feedback_drawer.mini_tab.aria_close": { en: "Close feedback panel" },
  "feedback_drawer.mini_tab.aria_open": { en: "Send feedback" },
  "feedback_drawer.popup_blocked": { en: "Link copied to clipboard — paste it in a tab to open GitHub." },
  "feedback_drawer.rate_limit": { en: "Thanks, 5 feedback submissions were already sent in the last hour." },
  "feedback_drawer.similar.label": { en: "A similar issue may already exist:" },
  "feedback_drawer.submit": { en: "Open on GitHub" },
  "feedback_drawer.submit_note": { en: "Redirecting to GitHub to finalize the submission. An automated analysis will enrich the issue." },
  "feedback_drawer.title": { en: "Send feedback" },
  "feedback_drawer.type.aria": { en: "Feedback type" },
  "feedback_drawer.type.bug": { en: "Bug" },
  "feedback_drawer.type.idea": { en: "Idea" },
  "feedback_drawer.type.question": { en: "Question" },
} as const

export type FeedbackDrawerManifestKey = keyof typeof feedbackDrawerManifest
