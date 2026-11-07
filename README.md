# backend

<div align="center">

[![Build Status](https://img.shields.io/github/checks-status/srlmgr/backend/main?color=black&style=for-the-badge&logo=github)][github-actions]
[![Security: bandit](https://img.shields.io/badge/Security-GoSec-lightgrey?style=for-the-badge&logo=springsecurity)](https://github.com/securego/gosec)
[![Dependencies Status](https://img.shields.io/badge/Dependencies-Up%20to%20Date-brightgreen?style=for-the-badge&logo=dependabot)][dependabot-pulls]
[![Semantic Versioning](https://img.shields.io/badge/versioning-semantic-black?style=for-the-badge&logo=semver)][github-releases]
[![Pre-Commit Enabled](https://img.shields.io/badge/Pre--Commit-Enabled-blue?style=for-the-badge&logo=pre-commit)][precommit-config]
[![License](https://img.shields.io/github/license/srlmgr/backend?color=red&style=for-the-badge)][project-license]
[![Go v1.27](https://img.shields.io/badge/Go-%20v1.27-black?style=for-the-badge&logo=go)][gomod-file]

Backend for SimRacing League Manager

</div>

## Persistent authentication sessions

OIDC sessions are stored in memory by default and are lost on restart. To store
them in the application's PostgreSQL database, first apply the database migrations
using `backend migrate`, then run the server with `--authn-persistent-sessions-enabled`.
The equivalent configuration key is `authn-persistent-sessions-enabled`; the environment
variable is `BACKEND_AUTHN_PERSISTENT_SESSIONS_ENABLED`.

The database stores OIDC access and refresh tokens. Restrict database access,
use encrypted database connections, and protect database backups and logs.
Session cookie values are stored only as SHA-256 hashes. Expired sessions are
rejected on lookup and removed in bounded batches by a background worker.
Each instance removes up to 1,000 expired sessions at startup and hourly.
If the session table is unavailable at startup, the server fails rather than
falling back to memory. Switching storage modes does not migrate existing
sessions, so users must log in again.

Persistent sessions can be used across server instances sharing the database,
but in-progress OIDC login state is still held in memory: callbacks must reach
the instance that initiated login. A restart during login also interrupts it.
Session persistence does not extend the browser cookie's lifetime. Token
rotation and a database commit cannot be atomic; if the identity provider
rotates a token but the database update fails, another login may be required.

[project-license]: ./LICENSE
[github-actions]: ../../actions
[github-releases]: ../../releases
[precommit-config]: ./.pre-commit-config.yaml
[gomod-file]: ../main/go.mod
[dependabot-pulls]: ../../pulls?utf8=%E2%9C%93&q=is%3Apr%20author%3Aapp%2Fdependabot
