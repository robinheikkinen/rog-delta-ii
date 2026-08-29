# Security policy

Please report security-sensitive findings privately through GitHub Security Advisories rather
than opening a public issue.

The HID recorder is intentionally read-only. Device-write support must not be added for
unverified commands: malformed vendor reports can reset, reconfigure, or potentially damage
hardware. Captures may also reveal device state and should be reviewed before publication.
