---
sidebar_position: 2
title: Contact Form
---

# Contact Form

webhull includes a built-in contact form that handles validation, spam filtering, rate limiting, and SMTP dispatch — no third-party form service required.

## How it works

```
Visitor fills form → POST /api/contact
  → honeypot check (reject bots instantly)
  → rate limit check (3 req / 15 min / IP)
  → field validation (name, email, subject, message)
  → spam filter (keywords, link count, character patterns)
  → SMTP dispatch to recipients
  → auto-reply to sender (optional)
  → JSON response → JS shows success/error message
```

## Enabling the form

The contact form is disabled by default. Enable it by adding a `contact:` block and configuring SMTP:

```yaml
contact:
  enabled: true
  recipientName: "Acme Support"
  recipients:
    - "hello@example.com"
  subject:
    en: "New message from {{.Name}}"
    de: "Neue Nachricht von {{.Name}}"
  maxLinks: 2               # messages with more links are rejected as spam
  rateLimit:
    requests: 3
    window: 15m

mail:
  host: "smtp.example.com"
  port: 587
  username: "${SMTP_USERNAME}"
  password: "${SMTP_PASSWORD}"
  from: "noreply@example.com"
  fromName: "Acme Corp"
  useTLS: true
```

The `contact` template automatically renders the form when this is configured. No template changes needed.

## Form fields

The form collects:

| Field | Validation |
|-------|-----------|
| Name | Required, non-empty |
| Email | Required, valid format, non-disposable domain |
| Subject | Required, non-empty |
| Message | Required, non-empty |
| Website | Honeypot — must be empty; any value triggers instant rejection |

## Rate limiting

Submissions are rate-limited per IP address. Defaults: 3 requests per 15-minute window. Configure via `contact.rateLimit`:

```yaml
contact:
  rateLimit:
    requests: 5
    window: 30m
```

## Spam filter

The spam filter rejects messages that match any of the following:

- Known spam keywords (`viagra`, `casino`, `crypto lottery`, etc.)
- Banned phrases (`click here`, `free money`, etc.)
- More than `maxLinks` URLs in the message body
- High uppercase character ratio (>40%)
- Low word variety (repetitive text)
- Repeated character sequences

Custom spam keywords can be added programmatically — see the `forms` package.

## Auto-reply

Send a confirmation email to the visitor after a successful submission:

```yaml
mail:
  templates:
    en:
      subject: "We received your message"
      body: |
        <p>Hi {{.Name}},</p>
        <p>Thank you for getting in touch. We will reply within one business day.</p>
        <p>Acme Corp</p>
    de:
      subject: "Ihre Nachricht ist angekommen"
      bodyFile: "mail/auto-reply-de.html"   # load from file instead of inline
```

`bodyFile` is resolved relative to the config file directory. The template variables available are `{{.Name}}`, `{{.Email}}`, `{{.Subject}}`, `{{.Message}}`.

## Customising form labels

Form labels default to built-in DE/EN strings. Override any field per language via `ui.contactForm`:

```yaml
ui:
  en:
    contactForm:
      heading: "Send us a message"
      nameLabel: "Your name"
      emailLabel: "Email address"
      subjectLabel: "Subject"
      messageLabel: "Message"
      submitText: "Send"
      successMsg: "Thank you! We will be in touch."
      successRefMsg: "Your reference:"
      errorMsg: "Something went wrong. Please try again."
      requiredMsg: "Please fill in all required fields."
  de:
    contactForm:
      heading: "Schreib uns"
      submitText: "Absenden"
```

Only non-empty values override the built-in defaults — you do not need to set every field.

## Using the contact template

Add a page with `template: contact` in your content directory:

```html
---
id: contact
template: contact
title: "Contact"
description: "Get in touch."
heroLine1: "Let's talk."
heroSubtitle: "No pitch decks required."
---

<p>Fill in the form and we will get back to you within one business day.</p>
```

The body content (above) appears above the form. Leave it empty if you want just the form with no intro.

## SMTP with TLS

For SMTP servers that require TLS on connection (port 465 rather than STARTTLS on 587):

```yaml
mail:
  host: "smtp.example.com"
  port: 465
  useTLS: true
```

For STARTTLS (port 587, the default), `useTLS: false` (or omit — default is `false`; the connection upgrades automatically).
