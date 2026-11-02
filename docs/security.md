# Security

This describes what the code does today. KeyDepot is in beta and hasn't had an independent security review.

## How keys are encrypted

A random 256-bit **data key** encrypts every private key. Each protection mode only changes how that data key is kept.

```
Private key ── AES-256-GCM ── data key
                                 │
             ┌───────────────────┼─────────────────────┐
             │                   │                     │
       Password mode        Password mode        Touch ID / No prompt
      wrapped by password   wrapped by code     stored in the Keychain
         (Argon2id)           (Argon2id)
```

- **Sealing:** AES-256-GCM with a random nonce for each write. The item's name is the additional data, so a sealed key can't be swapped for another without detection. Code: [`lib/crypt`](../lib/crypt/crypt.go).
- **Key derivation:** Argon2id with 3 passes, 64 MiB of memory, 4 threads and a random 16-byte salt.
- **Recovery code:** 26 random base32 characters, about 130 bits. It's wrapped the same way as the password, under a separate role, so a wrapped password key can't be used as a wrapped recovery key.
- **Changing modes:** moving into password mode generates a new data key and re-encrypts every item in one write of the vault, so a data key the Keychain held no longer opens it. Changing the password re-wraps the same data key, so the recovery code keeps working.

## What the modes protect against

| Threat | Password | Touch ID or Mac password | No extra prompt |
|---|---|---|---|
| Someone copies only the vault file | Protected, depending on password strength | Protected | Protected |
| Someone walks up to your unlocked Mac while KeyDepot is locked | Protected | Protected by the prompt | **Not protected** |
| Malware running as you | **Not protected** once you unlock | **Not protected** | **Not protected** |

The Touch ID or Mac password mode is only on macOS. Its prompt is a gate in the app, not encryption. The data key sits in your login Keychain, and the prompt decides whether KeyDepot uses it. Password mode is the only one where opening the vault needs something that isn't stored on your computer.

Editing the vault file can't lower the protection. A password-wrapped key in the file always means password mode, and in the other modes the Keychain item decides the mode, not the file. A vault with keys and no Keychain item shows an error rather than opening.

## What isn't protected

- **`keys.json` is unencrypted.** Key names, types, projects and dates are readable by anything that can read your files, so the app can list them while locked. Don't put secrets in a key's name.
- **Memory:** locking clears the data key, but Go may keep other copies in memory. KeyDepot doesn't defend against memory dumps. Revealed and decrypted keys also pass through memory as ordinary strings.
- **Clipboard:** other apps can read a copied key during the 60 seconds, clipboard history tools may keep it, and it isn't cleared if KeyDepot quits first.
- **Tampering by removal:** each item is authenticated on its own, but the vault as a whole isn't. Someone who can write to the file can delete items or restore an older copy without KeyDepot noticing. They still can't read or forge keys.
- **Password strength:** the minimum is 8 characters, or 12 for exports. Argon2id slows guessing but can't save a weak password.
- **Other apps and the secret store:** other apps you run may be able to ask for the data key in the modes that keep it there. macOS decides whether to prompt you, and on Linux they may be able to read it without asking.
- **Windows and Linux:** these builds are untested, and the design hasn't been checked against how those systems protect files and secrets. For example, the `0600` file permission means nothing on Windows.

## Exports

An export has its own random data key, wrapped by its own password, so it reveals nothing about the vault's key. The key list inside is encrypted too.

Imported files are treated as untrusted. KeyDepot rejects files over 64 MiB and unknown format versions, and limits the Argon2id costs a file can ask for, all before deriving a key. Every key must decrypt before any is returned.

## Updates

Updates are signed with Ed25519. The app holds the public key, and refuses any update whose signature doesn't match. The private key lives outside the repository, in a password manager and in a CI secret. Anyone who gets it could ship code to every installed app, if they could also publish to the release location.

Beta builds aren't signed with an Apple Developer ID or a Windows code signing certificate, so your system can't verify who made the first download you install. An update may also make macOS ask again for access to the Keychain item.

## File format

The vault and exports share one JSON format, written with `0600` permissions on macOS and Linux:

```json
{
  "version": 1,
  "mode": "password",
  "password": { "salt": "…", "kdf": { "time": 3, "memory": 65536, "threads": 4 }, "sealed": "…" },
  "recovery": { "salt": "…", "kdf": { "time": 3, "memory": 65536, "threads": 4 }, "sealed": "…" },
  "items": { "key.<id>": "…" }
}
```

Binary values are base64. Sealed values are the nonce followed by the ciphertext and tag. `mode` is only for display before unlocking. An export has no `recovery` or `mode`, and holds its key list under the item `index`.

## Reporting a problem

Please use GitHub's [private vulnerability reporting](https://github.com/tbdsux/keydepot/security/advisories/new) rather than a public issue.
