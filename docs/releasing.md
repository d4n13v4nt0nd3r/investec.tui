# Releasing

How to publish the downloadable Mac and Windows packages.

Releases are built from a maintainer's Mac rather than CI, because signing
needs the Developer ID certificate in the local keychain and we would rather
not export it into GitHub secrets.

## What a release produces

| Artifact | Platform | Notes |
|----------|----------|-------|
| `InvestecTUI-<version>.dmg` | macOS | Universal (Intel + Apple Silicon), signed, notarized, stapled |
| `InvestecTUI-<version>-Windows-x64.exe` | Windows | Unsigned, the download most people want |
| `InvestecTUI-<version>-Windows-ARM64.exe` | Windows on ARM | Unsigned, optional |
| `SHA256SUMS.txt` | -- | Checksums for all of the above |

The repository is internal, so release assets are only downloadable by signed-in
BeamMoney org members.

## Running a release

```bash
./release v1.0.0
```

Useful variations while testing:

```bash
./release v1.0.0 --skip-notarize --no-publish   # full local dry run
./release v1.0.0 --no-publish                   # sign and notarize, do not upload
```

The script refuses to publish a build that was not notarized.

Everything lands in `dist/`, which is gitignored.

## One-time setup

Do these once. After that `./release` needs no further input.

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

## Optional: app icon

If `packaging/macos/AppIcon.icns` exists, the release script bundles it as the
app icon. Without it macOS shows the generic application icon. To create one
from a 1024x1024 PNG:

```bash
mkdir -p /tmp/AppIcon.iconset
for size in 16 32 64 128 256 512; do
  sips -z $size $size icon.png --out /tmp/AppIcon.iconset/icon_${size}x${size}.png
  sips -z $((size*2)) $((size*2)) icon.png --out /tmp/AppIcon.iconset/icon_${size}x${size}@2x.png
done
iconutil -c icns /tmp/AppIcon.iconset -o packaging/macos/AppIcon.icns
```

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
