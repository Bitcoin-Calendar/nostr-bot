# Bitcoin Calendar Nostr bot

Publishes English events for today's month and day as **kind 1 notes only**.
Each note contains the title, description, English event-page URL and, when
available, the first explicit image URL. Archive references remain on the site.
It never publishes kind 20 or long-form articles. The older kind 20 helpers in
`internal/nostr` are not called by this entrypoint.

## Build and verify

Requires Go 1.24.1 or newer.

```sh
go test ./...
go vet ./...
go build -o nostr_bot .
./nostr_bot --version
```

Load the variables in `deploy/nostr-bot.env.example` through your scheduler.
The API uses `X-API-Key` and `/api/events?month=MM&day=DD&lang=en&limit=1000`.
The restored API's date is `YYYY-MM-DD`; media and references are JSON arrays
encoded as strings, or null. Missing or malformed event arrays fail the run.

## Modes

| Command | Behavior |
| --- | --- |
| `nostr_bot` | Publish today's events, then exit. Requires the private key. |
| `nostr_bot --preview --date 2026-10-04` | Print unsigned posts for that date. No private key needed, no relay writes or state changes. |
| `nostr_bot --probe` | Check API, formatting, state lock and relay reads. No private key needed and no posts sent. |
| `nostr_bot --check` | Also validate the private key and display its public npub. No posts sent. |
| `nostr_bot --version` | Display build identity without loading configuration. |

Unknown flags, positional arguments and combined modes fail. Date overrides
are allowed only with `--preview`, never when publishing. A read-only check does
not prove that relays will accept this account's writes.

## Configuration

| Variable | Default / purpose |
| --- | --- |
| `BOT_API_ENDPOINT` | `http://127.0.0.1:3000/api` |
| `BOT_API_KEY` | Required API key |
| `NOSTR_PRIVATE_KEY_EN` | Required for posting and `--check`; 64-character hex or `nsec` |
| `NOSTR_RELAYS` | Primal, Damus, nos.lol and Snort; comma-separated WebSocket URLs |
| `WEB_BASE_URL` | `https://bitcoin-calendar.org` |
| `BOT_STATE_DIR` | `/var/lib/bitcal-nostr` |
| `BOT_TIMEZONE` | `America/New_York`; determines the current day |
| `BOT_START_DATE` | `2026-10-03`; publishing before this date is skipped |
| `BOT_SLEEP_SECONDS` | `1800`; pause between attempted events, never after the last |

The bot does not read `.env` automatically. systemd loads the protected file.
Do not pass private keys on the command line or commit them.

## Delivery and restarts

The process locks its state directory, preventing overlapping runs. It saves a
signed event before publishing and reuses that exact event after an ambiguous
failure, so retransmission retains the same Nostr ID. After at least one relay
acknowledges acceptance it marks the event delivered and skips it on subsequent
runs for that account and calendar date. Partial relay failure is reported in
the journal; an all-relay failure exits nonzero. Successful notes are not
replayed merely to repair delivery to one failed relay.

Do not delete the state directory: it is the publication history. A failure to
read or validate saved state stops the run rather than creating a replacement
post. A later year's anniversary has its own publication date and is posted
again as intended. State currently has no automatic pruning.

## Server deployment and activation

Use `deploy/bitcal-nostr.service` and `deploy/bitcal-nostr.timer`. The service
runs as `bitcal`, owns `/var/lib/bitcal-nostr`, logs to the journal and does not
restart automatically. The timer runs at **08:00 America/New_York**, including
DST changes. `Persistent=false` skips missed runs rather than posting late.
For the initial rollout, leave the timer disabled until the account is checked.
Set `BOT_START_DATE` to the intended first publication date before enabling the timer.

On the server, edit the prepared root-only configuration:

```sh
sudo nano /etc/bitcal/nostr-bot.env
# Set NOSTR_PRIVATE_KEY_EN=<your hex or nsec key>, keeping the other settings.
sudo /srv/bitcal/bots/nostr/check.sh
```

Verify the displayed **Account: npub...** is the intended account, then enable
only the timer:

```sh
sudo systemctl enable --now bitcal-nostr.timer
systemctl list-timers bitcal-nostr.timer --no-pager
```

Do not start the service manually to validate it: that publishes real posts.
After the scheduled run, inspect `journalctl -u bitcal-nostr.service` for relay
acknowledgements and the published event ID.

The activation helper `check.sh` uses a transient unit with the same sandbox
and protected environment as the posting service, but passes `--check`.

The historical Docker instructions under `docs/` describe the previous runner;
use this README for the kind 1 deployment. The separate `long-form-bot`
repository is outside this service.
