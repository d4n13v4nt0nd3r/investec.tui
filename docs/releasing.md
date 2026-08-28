# Releasing

How to publish the downloadable Mac and Windows packages.

Releases are built from a maintainer's Mac rather than CI, because signing
needs the Developer ID certificate in the local keychain and we would rather
not export it into GitHub secrets.

## What a release produces

| Artifact | Platform | Notes |
|----------|----------|-------|
| `InvestecTUI-<version>.dmg` | macOS | Universal (Intel + Apple Silicon), signed and notarized when a certificate is available |
| `InvestecTUI-<version>-Windows-x64.exe` | Windows | Unsigned, the download most people want |
| `InvestecTUI-<version>-Windows-ARM64.exe` | Windows on ARM | Unsigned, optional |
| `SHA256SUMS.txt` | -- | Checksums for all of the above |

The repository is internal, so release assets are only downloadable by signed-in
BeamMoney org members.

## Releasing without an Apple certificate

This is where we are today: nobody on the team has been added to the company
Apple Developer account yet, so there is no Developer ID certificate to sign
with and nothing to notarize.

```bash
./release v1.0.0 --unsigned
```

That builds and publishes exactly the same artifacts, with two differences:

- The `.app` gets an **ad-hoc** signature instead of a Developer ID one. This
  is not a mark of who built it; it exists because Apple Silicon refuses to
  run a Mach-O with no signature at all, and because it seals the bundle so
  the `Info.plist` and icon cannot be swapped without breaking it.
- The `.dmg` is not signed and not notarized, so Gatekeeper stops the first
  launch. Users get past it with Control-click -> **Open**, or **Open Anyway**
  in System Settings -> Privacy & Security. The release notes and the README
  both spell this out.

The Mac download is therefore no worse off than the Windows one, which has
always been unsigned. Once the Developer ID certificate exists, complete the
one-time setup below and drop the flag: nothing else about the release
changes.

## Running a signed release

```bash
./release v1.0.0
```

Useful variations while testing:

```bash
./release v1.0.0 --skip-notarize --no-publish   # full local dry run
./release v1.0.0 --no-publish                   # sign and notarize, do not upload
./release v1.0.0 --unsigned --no-publish        # build the unsigned packages only
```

The script refuses to publish a **signed** build that skipped notarization,
since that is a step someone meant to run rather than a decision. `--unsigned`
is a decision, so it publishes.

Everything lands in `dist/`, which is gitignored.

## One-time setup

Not needed for `--unsigned` releases. Do these once when the Apple Developer
account is available, and after that `./release` needs no further input.

### Part A -- Developer ID Application certificate

Requires the **Account Holder** role on the Apple Developer team. Admins can
only do this if they have been granted the cloud-managed Developer ID access
role. You may create up to five of these certificates.

1. Open **Keychain Access** (Cmd+Space, type `Keychain Access`).
2. Menu bar -> **Keychain Access** -> **Certificate Assistant** -> **Request a
   Certificate From a Certificate Authority...**
3. Fill in **User Email Address** (your Apple Developer account email) and
   **Common Name** (for example `Daniel - Investec TUI`). Leave **CA Email
   Address** empty.
4. Select **Saved to disk**, tick **Let me specify key pair information**, then
   click **Continue**.
5. Save it as `CertificateSigningRequest.certSigningRequest` in `Downloads`.
6. Key Size **2048 bits**, Algorithm **RSA**, click **Continue**, then **Done**.
7. Go to https://developer.apple.com/account -> **Certificates, Identifiers &
   Profiles** -> **Certificates**.
8. Click the **+** button at the top left.
9. Under the **Software** heading select **Developer ID**, click **Continue**.
10. Choose **Developer ID Application**. Do not choose Developer ID Installer:
    we ship a DMG, not a `.pkg`. Click **Continue**.
11. If asked for a profile type, choose **G2 Sub-CA (Xcode 11.4.1 or later)**.
12. Click **Choose File**, pick the `.certSigningRequest` from step 5, then
    click **Continue**.
13. Click **Download**. A `.cer` file lands in `Downloads`.
14. Double-click the `.cer` to install it. It appears under **My Certificates**
    in Keychain Access.
15. Verify:

    ```bash
    security find-identity -v -p codesigning
    ```

    You should see a line reading `"Developer ID Application: <Team Name> (TEAMID)"`.
    Note the 10-character Team ID, you need it in Part C. The release script
    finds this identity by itself, so nothing needs to be configured.

### Part B -- App-specific password for notarization

Simpler than an App Store Connect API key and needs no extra roles.

1. Go to https://appleid.apple.com and sign in.
2. **Sign-In and Security** -> **App-Specific Passwords** -> **+**.
3. Name it `notarytool investec-tui`, click **Create**, and copy the
   `xxxx-xxxx-xxxx-xxxx` password. It is shown only once.

### Part C -- Store the notarization credentials

Run this once, replacing the two placeholders. It prompts for the app-specific
password from Part B, so the secret is never typed on the command line:

```bash
xcrun notarytool store-credentials "investec-tui-notary" \
  --apple-id "<your-apple-id-email>" \
  --team-id "<TEAMID>"
```

The profile name `investec-tui-notary` is what `scripts/release.sh` looks for.
No Apple secrets are stored in this repository.

## App icon

Both platforms ship the zebra mark in white on the app's own background colour.
Every artifact is committed, so a normal release does not rebuild any of them:

| Artifact | Used by |
|----------|---------|
| `packaging/source/tui-zebra-logo.png` | The original artwork everything derives from |
| `packaging/macos/AppIcon.icns` | Copied into the `.app` by `scripts/release.sh` |
| `packaging/windows/app.ico` | Source for the resource objects below |
| `rsrc_windows_amd64.syso`, `rsrc_windows_arm64.syso` | Linked into the `.exe` automatically, because `go build` picks up any `*.syso` sitting next to the main package |

The `.syso` files must stay at the repository root. Go matches them by the
`_windows_amd64` / `_windows_arm64` filename suffix, so each is only linked into
its own target and they are ignored entirely on macOS builds.

### The DMG's icon

`scripts/release.sh` also gives the mounted volume itself a custom icon (the
drive shown in the Finder window that opens when someone double-clicks the
`.dmg`), using the same zebra mark. This needs a writable image first: the
custom-icon flag is a bit in the HFS+ catalog entry for the volume's root
directory, not a plain file attribute, so it can only be set once there is an
actual mounted volume to set it on -- setting it on the plain folder being
packaged has no effect. The script therefore builds an uncompressed image,
mounts it, drops `.VolumeIcon.icns` at the root and flags the root directory,
then unmounts and converts to the compressed format that ships.

What this deliberately does not attempt is a custom icon on the `.dmg` file
itself, i.e. what Finder shows in Downloads before it is ever double-clicked.
That icon is stored as a resource fork plus a Finder-info flag on the outer
file, which macOS keeps as an extended attribute rather than file content.
Extended attributes do not survive a plain HTTP download -- GitHub Releases,
`curl`, browsers, etc. only transfer the data fork -- so setting one would only
ever be visible on the machine that built it, not to anyone who downloads the
release. This is a real limitation of how macOS stores that particular kind of
icon, not something a build script can work around. The Windows `.exe` icon
does not have this problem because it is embedded directly in the PE file's
own bytes, which is exactly why `.exe` downloads show the zebra immediately
while the `.dmg` download will always show the generic disk-image icon until
it is mounted.

### Rebuilding

Only needed after changing the artwork or the icon geometry:

```bash
./icons
```

This needs Pillow (`python3 -m pip install Pillow`) and has to run on macOS,
since `iconutil` builds the `.icns`. It regenerates everything, then links a
throwaway Windows binary per architecture and reads the icon back out of the
PE, because a malformed resource object links without complaint and simply
produces an executable with no icon.

Afterwards check `packaging/source/preview.png`, a contact sheet of both
platforms at the sizes that actually get used.

### Why small sizes look different

At or below 24px the icon switches to a solid head silhouette instead of the
striped mark. The artwork has roughly a dozen stripes, which cannot be
represented in a 16px-wide shape at all -- they average out to flat grey no
matter how the resampling is done. The crossover is `SIMPLIFY_AT` in
`packaging/tools/build_icons.py`.

## Troubleshooting

**`No "Developer ID Application" certificate found`**
Part A has not been completed, or the certificate has expired. Check with
`security find-identity -v -p codesigning`.

**Notarization returns `Invalid`**
Fetch the detailed reasons:

```bash
xcrun notarytool log <submission-id> --keychain-profile "investec-tui-notary"
```

The usual cause is a binary signed without the hardened runtime. The release
script passes `--options runtime`, so this should not happen unless the signing
steps were changed.

**Users still see a Gatekeeper warning**
Confirm the ticket was stapled:

```bash
xcrun stapler validate dist/InvestecTUI-<version>.dmg
spctl --assess --type open --context context:primary-signature -vv dist/InvestecTUI-<version>.dmg
```

## Signing the Windows build

The Windows `.exe` files are unsigned, so first-time users see a SmartScreen
prompt once. Removing it needs an Authenticode certificate. Azure Trusted
Signing is the cheapest route and would slot in as an extra step in
`scripts/release.sh` after the Windows builds, without changing anything else.
