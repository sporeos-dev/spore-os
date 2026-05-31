# Contributing to Spore OS

Thank you for your interest in contributing to **Spore OS**! We appreciate your help in making this project better.

## Licensing Model

Spore OS will be **dual-licensed** to support both the open-source community and sustainable development.
- **Open Source License**: Contributions are primarily licensed under the [Insert Primary License, e.g., AGPL-3.0].
- **Commercial License**: The project owner retains the right to re-license contributions under proprietary terms for commercial use.

To enable this flexibility, **all contributors must sign a Contributor License Agreement (CLA)**. This ensures that the project maintainers have the necessary rights to distribute the code under multiple licenses without legal ambiguity.

## Signing the CLA

We use an automated system to manage CLAs. You do not need to download, print, or email any documents.

1.  **Open a Pull Request (PR)**: Submit your changes as usual.
2.  **Wait for the Bot**: The **CLA Assistant** bot will automatically comment on your PR.
3.  **Review and Sign**:
    - Click the link in the bot's comment to view the [Individual Contributor Non-Exclusive License Agreement](https://gist.github.com/mharr-cmpt/8bec7c34912633cbd1ebca5e497b780d).
    - Click the **"I Agree"** button to digitally sign the agreement.
4.  **Merge**: Once signed, the bot will update the status, and your PR will be eligible for merging.

> **Note**: If you are contributing on behalf of an employer or organization, please ensure your employer has approved this agreement. If you are not the sole copyright owner of the contribution, all co-authors must also sign.

## Branching Strategy

- **`main`** — stable, clean integration. Only receives merges from `develop` branches via pull request.
- **`develop`** — active integration branch. Day-to-day work lands here. Expected to be messy at times.
- **`release/x.x.x`** — cut from `main` when preparing a release. Only bug fixes go in at this stage.
- **Feature branches** — branch from `main`, merge back to `develop` via pull request.

CI runs on every push to `develop`, `main`, and `release/**`. Pull requests targeting `main` or `release/**` additionally run static analysis before merge.

On **Windows**, a Unix-compatible shell is required to run the project scripts (e.g. [Git Bash](https://git-scm.com/downloads)).

## Development Guidelines

- **Code Style**: Follow the existing coding conventions in the repository.
- **Testing**: Ensure all tests pass before submitting a PR. Run `make check` && `make analyze` locally before pushing.
- **Commits**: Write clear, descriptive commit messages.
- **Issues**: If you find a bug or have a feature request, please open an issue first to discuss it.

## Questions?

If you have questions about the CLA or the contribution process, please open an issue or contact the maintainers directly.

Thank you for helping build Spore OS!