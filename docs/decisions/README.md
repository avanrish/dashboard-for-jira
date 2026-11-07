# Architecture decision records

This directory records the decisions that shape the project: what was decided, why, and what else was considered.
The code shows what the project does; these records explain why it is that way.
They exist mainly for AI coding agents and new contributors, who see the code but not the reasoning behind it, and might otherwise undo deliberate choices or propose options that were already rejected.

## When to write one

Write a record when a decision is hard to reverse, or when a reader could not work out the reasoning from the code alone, such as choosing a library, a protocol, a storage approach, or deliberately not doing the obvious thing.
Skip it for choices that are cheap to change or self-evident from the code.

## Format

- One file per decision, named `NNNN-short-title.md` with the next free number.
- Start from [`template.md`](template.md) and keep it short: a reader should finish in a couple of minutes.
- Status is one of `Proposed`, `Accepted`, `Accepted, amended by NNNN`, `Rejected`, `Deprecated`, or `Superseded by NNNN`.

## Changing a decision

Once a record has been pushed, do not rewrite it; until then, edit it freely.
To replace a decision, write a new record, then set the old one's status to `Superseded by NNNN` and link the new one.
To change only part of a decision, write a new record that says what it changes, then set the old one's status to `Accepted, amended by NNNN` and link the new one.
Fixing typos or adding links is fine.
