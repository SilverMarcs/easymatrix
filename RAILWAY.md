# Railway deployment

EasyMatrix already ships as a two-stage Docker container. Use one long-running
service in Railway's Singapore region with one persistent volume. Do not enable
replicas: Railway volumes and the SQLite/gomuks session are single-writer.

## Security boundary

Railway encrypts customer data at rest and restricts staff access, but this is
not confidential computing. EasyMatrix must decrypt Matrix events to serve the
API and send notifications, so plaintext exists in its database and process
memory. A Railway or infrastructure administrator with sufficient privilege
can technically access it. Containerization, a volume key, SQLCipher, or a
secret stored in Railway variables cannot prevent the host from inspecting a
running process.

If the requirement is that the hosting provider must be technically unable to
read plaintext, do not deploy the current architecture to Railway. Keep the
decrypting server on hardware you control, use a confidential-computing host
with remote attestation, or redesign Relay so decryption occurs only on the
Apple device.

## Service setup

1. Deploy this repository using the root `Dockerfile`.
2. In **Settings > Scale**, select **Southeast Asia (Singapore)** and keep one
   instance.
3. Attach one volume at `/data`. Railway automatically provides
   `RAILWAY_VOLUME_MOUNT_PATH`; EasyMatrix resolves it to `/data/gomuks`.
4. Set `RAILWAY_RUN_UID=0`. Railway mounts volumes as root, so its documented
   non-root-container workaround is required for this image.
5. Generate a public domain. Keep serverless disabled because Matrix sync,
   WebSockets, and APNs delivery require a continuously running process.
6. Set a memory limit of 512 MB initially. The observed steady-state process is
   much smaller, but sync bursts and attachment handling need headroom.

The image sets `EASYMATRIX_EPHEMERAL_ROOT=/tmp/easymatrix`. Only gomuks
`config/` and `data/`, OAuth state, and push-device registrations remain on the
volume. Downloaded/decrypted media, upload staging, logs, and caches disappear
on redeploy. Successfully sent uploads are removed immediately.

## Required secrets

Set these as Railway service variables; never commit their values:

```text
MATRIX_ACCESS_TOKEN=<random API bearer token used by Relay>
EASYMATRIX_MANAGE_SECRET=<different random secret for /manage>
MATRIX_ALLOW_QUERY_TOKEN=false
```

For production push notifications, also set:

```text
APNS_KEY=<contents of the Apple .p8 key>
APNS_KEY_ID=<Apple key ID>
APNS_TEAM_ID=<Apple team ID>
APNS_TOPIC=<Relay bundle identifier>
APNS_ENVIRONMENT=production
```

`APNS_KEY` accepts a multiline PEM value or a single line whose newlines are
written as `\n`. Do not configure `MATRIX_USERNAME`, `MATRIX_PASSWORD`,
`MATRIX_LOGIN_TOKEN`, or `MATRIX_RECOVERY_KEY` when migrating an existing
session; the migrated state already contains the Matrix device credentials.

## Seamless session migration

Do not upload only `gomuks.db`. Stop the local EasyMatrix process first so the
SQLite database, WAL, crypto store, and config form one consistent snapshot.
Then upload only the `config` and `data` directories from the existing gomuks
root:

```sh
railway login
railway link
railway volume add --mount-path /data
railway volume files upload /absolute/path/to/gomuks/config /gomuks/config
railway volume files upload /absolute/path/to/gomuks/data /gomuks/data
```

If the volume already exists, omit `railway volume add`. Do not migrate
`cache/`, `logs/`, `assets/`, or `api-uploads/`; all of those can be rebuilt or
discarded. Start or redeploy the service only after both directory uploads
finish.

With a successful state migration, no new Matrix login is needed. Relay still
authenticates to EasyMatrix using the new `MATRIX_ACCESS_TOKEN`, so update the
app's server URL and bearer token. If migration is skipped, open
`/manage?secret=<EASYMATRIX_MANAGE_SECRET>` and perform a fresh Beeper login and
verification; that creates a new Matrix device and may require the recovery
key.
