# Messaging help and CLI

This feature remains on integration HOLD. These instructions describe the isolated module/harness, not currently enabled Circula functionality.

## Member workflow

Open `/conversations` on the explicitly mounted host, choose a member ID, optional local trade/listing reference, and retention days. Start the conversation. The recipient must still be active in the same network. The server validates any reference and shorter network retention policy. Messages never create or accept trades.

Open a conversation, type literal text and submit **Send message**. All essential actions work without JavaScript. Reload to see new messages; the inbox displays persisted unread counts. Use the last visible sequence with **Mark read**. Longer histories have sequence pagination. Message text is escaped, with no remote images, link previews or attachments.

**Hide current history from my view** hides only your existing copy; the other participant retains theirs. A later new message reappears in your inbox. **Close conversation for sending** stops both participants from sending new messages while retaining readable history. The inbox block form supports block and unblock; either direction blocks new conversations and messages while leaving historical access subject to membership/retention.

Report another participant's visible message with a reason. Duplicate reports preserve the first reason. Reports are reviewed separately by authorized moderators. Sending more messages does not resolve a report. A moderator can place a legal hold; held content is still hidden from ordinary members after its inactivity deadline.

Messages can be read by the hosting operator. This is not end-to-end encrypted. Default expiry is 30 days after the latest accepted message; a network may shorten this. Reading, reporting and read markers do not extend retention. Export downloads one page of your currently visible messages as plaintext JSON. Fetch another page using the final sequence in `?after=...`; protect downloaded copies yourself.

## Standalone HTTP client

Build from this directory with Go 1.26.11:

```sh
go build -o /tmp/circula-messages ./cmd/circula-messages
```

Set `CIRCULA_TOKEN` to a valid host-issued non-cookie API credential through your normal secret-management mechanism. The CLI never prints it or follows redirects. Specify a real HTTPS origin and authorized network. Options precede the command:

```sh
circula-messages -url https://circula.example -network community list
circula-messages -url https://circula.example -network community -target bob -key create-example-01 create
circula-messages -url https://circula.example -network community -thread CONVERSATION_ID read
printf '%s' 'Meeting time?' | circula-messages -url https://circula.example -network community -thread CONVERSATION_ID -key message-example-01 send
circula-messages -url https://circula.example -network community -thread CONVERSATION_ID -after 12 read-marker
circula-messages -url https://circula.example -network community -target bob block
circula-messages -url https://circula.example -network community -target bob unblock
printf '%s' 'Unwanted solicitation' | circula-messages -url https://circula.example -network community -thread CONVERSATION_ID -after 12 report
circula-messages -url https://circula.example -network community -thread CONVERSATION_ID -after 100 export
circula-messages -url https://circula.example -network community -thread CONVERSATION_ID hide
circula-messages -url https://circula.example -network community -thread CONVERSATION_ID close
```

Use a new stable key for each distinct create/send request and reuse that same key on retry. A conflict means the key was already used for different content. Read message/report text from stdin; avoid placing private text in shell command arguments/history in real use. Successful output is JSON. Non-success output contains only the HTTP status; it does not echo remote error bodies, message content or credentials. The client bounds input and response size and requires HTTPS.

The moderator and retention APIs in `CONTRACT.md` require separate host permissions. They have no implicit default administrator account. No background process, maintenance schedule or network listener starts from this package. See the contract for secure backup/restore and disabling procedures.

## Moderator and maintenance commands

These routes and commands require the separately granted network permissions described in CONTRACT.md. Open `/conversations/CONVERSATION_ID/moderation` for report disposition and legal holds, or `/messaging-maintenance` for retention maintenance. They work without JavaScript. A hold must exist before expiry; an expired released hold cannot be reopened. Shortening existing retention can make old content unavailable immediately and never extends an existing deadline.

```sh
circula-messages -url https://circula.example -network community -cursor LAST_CONVERSATION_ID list
circula-messages -url https://circula.example -network community -thread CONVERSATION_ID reports
circula-messages -url https://circula.example -network community -thread CONVERSATION_ID held
circula-messages -url https://circula.example -network community -thread CONVERSATION_ID -moderation-action hold moderate
circula-messages -url https://circula.example -network community -days 1 retention
circula-messages -url https://circula.example -network community purge
```

For report paging, pass the final sequence in `-after` and final reporter in `-reporter`. To resolve/dismiss one report, use `-moderation-action resolve` or `dismiss` with its `-after` and `-reporter`. To release a hold, use `-moderation-action release`. Never use downloaded plaintext outside the owner's or authorized reviewer's protected storage.

## Search private messages

Open a conversation and use **Search visible messages**. Search is literal and case-sensitive. `%`, `_`, quotes and non-English text remain ordinary characters. Search only finds messages still visible to you; hidden, expired and inaccessible conversations stay unavailable. Search does not mark messages read. Use **Next search results** for another page.

For the CLI, provide the phrase through stdin so it does not enter the process argument list:

```sh
printf '%s' 'meeting place' | circula-messages -url https://circula.example -network NETWORK -thread THREAD search
printf '%s' 'meeting place' | circula-messages -url https://circula.example -network NETWORK -thread THREAD -after 100 search
```

Queries are limited to 128 UTF-8 bytes. Neither browser nor CLI places search phrases in URLs.
