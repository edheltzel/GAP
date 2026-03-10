# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- PAI_SUPPRESS_VOICE environment variable support in VoiceGreeting hook to prevent duplicate voice greetings from programmatically-spawned Claude processes (e.g., `claude -p` invocations from Stop hooks or other handlers)

