# QDD Specification v1.0

## Core Concepts
- Certification is the single source of truth.
- Every bug becomes a Finding.
- Every Finding generates regression tests.
- Every closed Finding requires Evidence of Resolution.
- Public API contracts cannot change without approval.

## Lifecycle
Discovery
Analysis
Certification
Design
Testing
Implementation
Audit
Evidence
Quality Gate
Release

## Command Language

> Estado real (2026-07): AUDIT y CERTIFY son subcomandos de CLI reales. SPRINT y
> RELEASE existen solo como tools MCP (`qdd_sprint`, `qdd_release`) — se invocan
> pidiéndoselo a un IDE con IA, no tipeándolos en una terminal. FEATURE, FIX y
> WORLD-CLASS nunca se implementaron en ninguna forma; hoy esas intenciones se
> expresan como lenguaje natural vía `/qdd "..."` y las resuelve el IDE con IA
> conectado por MCP, no un comando dedicado. Ver docs/command-reference.md.

QDD AUDIT
QDD FEATURE
QDD FIX
QDD SPRINT
QDD CERTIFY
QDD RELEASE
QDD WORLD-CLASS
